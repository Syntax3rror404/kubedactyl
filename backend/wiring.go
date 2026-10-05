package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"app/api/v1alpha1"
	"app/internal/auth"
	"app/internal/bootstrap"
	"app/internal/clusterinfo"
	"app/internal/console"
	"app/internal/controller"
	"app/internal/diagnostics"
	"app/internal/egglibrary"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/httpapi"
	"app/internal/kube"
	"app/internal/schedule"
	"app/internal/selfupgrade"
	"app/internal/serverctl"
	"app/internal/settings"
	"app/internal/tenancy"
	"app/internal/users"
)

// kubeConfig loads the kubeconfig context (or the in-cluster service account); every client
// made from it shares the limiter.
func kubeConfig(kubeContext string, http1 bool, limiter *kube.RateLimiter) (*rest.Config, error) {
	restCfg, err := clientconfig.GetConfigWithContext(kubeContext)
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	restCfg.RateLimiter = limiter
	if http1 {
		// Proxies in front of the API server often cut long-running HTTP/2 streams with
		// a stream error, which client-go treats as a failure and answers with backoff.
		// Over HTTP/1.1 the same cut is a clean end of the watch and it reconnects at once.
		restCfg.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}
	return restCfg, nil
}

func newScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	adders := []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme,
		v1alpha1.AddToScheme,
		apiextensionsv1.AddToScheme,
	}
	for _, add := range adders {
		if err := add(scheme); err != nil {
			return nil, fmt.Errorf("building scheme: %w", err)
		}
	}
	return scheme, nil
}

// bootResult is what the start-up steps produce for the API.
type bootResult struct {
	signer     *auth.Signer
	setupToken string // "" once an administrator exists
}

// prepareCluster runs the start-up steps with a direct (uncached) client: install the CRDs,
// create the panel namespace, load the session key, create the first admin (or a setup token)
// and the panel settings.
func prepareCluster(
	ctx context.Context, restCfg *rest.Config, scheme *runtime.Scheme, cfg config, log *slog.Logger,
) (*bootResult, error) {
	ns := cfg.opts.Namespace
	direct, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	if err := bootstrap.InstallCRDs(ctx, direct); err != nil {
		return nil, fmt.Errorf("installing CRDs: %w", err)
	}
	if err := bootstrap.EnsureNamespace(ctx, direct, ns); err != nil {
		return nil, fmt.Errorf("creating namespace: %w", err)
	}
	// Installation scoping: this panel only handles its own user namespaces.
	tenancy.SystemNamespace = ns
	tenancy.NamespacePrefix = ns + "-user-"
	// In the cluster the panel may only write in user namespaces through its tenant role.
	tenancy.TenantRole, tenancy.PanelServiceAccount = cfg.tenantRole, cfg.serviceAccount
	signer, err := bootstrap.LoadSessionSigner(ctx, direct, ns)
	if err != nil {
		return nil, fmt.Errorf("loading session key: %w", err)
	}
	setupToken, err := bootstrap.Admin(ctx, direct, ns, "http://localhost"+cfg.addr, log.With("component", "bootstrap"))
	if err != nil {
		return nil, fmt.Errorf("creating initial admin: %w", err)
	}
	if err := settings.Bootstrap(
		ctx, direct, ns, cfg.opts.StorageClass, cfg.opts.LoadBalancerPool, log.With("component", "settings"),
	); err != nil {
		return nil, fmt.Errorf("creating panel settings: %w", err)
	}
	return &bootResult{signer: signer, setupToken: setupToken}, nil
}

// panel holds the controller manager (runs controllers and background jobs) and the API.
type panel struct {
	manager ctrl.Manager
	api     *httpapi.API
}

// newPanel builds the controller manager with all controllers and background runnables, the
// services the API uses and the API itself.
func newPanel(
	ctx context.Context, restCfg *rest.Config, limiter *kube.RateLimiter, scheme *runtime.Scheme, cfg config,
	boot *bootResult, devMode bool, log *slog.Logger,
) (*panel, error) {
	mgr, err := newManager(ctx, restCfg, scheme)
	if err != nil {
		return nil, err
	}
	kc, err := kube.New(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}
	hub := console.NewHub(kc, log.With("component", "console"))
	activity := files.NewActivity()
	rec, err := setupControllers(mgr, kc, hub, activity, cfg, log)
	if err != nil {
		return nil, err
	}
	upgradeCfg, updates, upgrader, err := setupSelfUpgrade(mgr, kc, cfg.upgrade, cfg.opts.Namespace, log)
	if err != nil {
		return nil, err
	}
	svc, err := setupServices(ctx, mgr, kc, hub, activity, rec, limiter, cfg, log)
	if err != nil {
		return nil, err
	}

	api := &httpapi.API{
		Client:      mgr.GetClient(),
		Reader:      mgr.GetAPIReader(),
		Kube:        kc,
		Hub:         hub,
		Files:       svc.files,
		Settings:    svc.settings,
		Users:       &users.Store{Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: cfg.opts.Namespace},
		Eggs:        svc.eggs,
		Library:     egglibrary.New(),
		Diagnostics: &diagnostics.Diagnoser{},
		Signer:      boot.signer,
		// Failed sign-ins: per client and user, per client (password spraying), per user.
		Limiter:        auth.NewLimiter(5, 5*time.Minute),
		ClientLimiter:  auth.NewLimiter(20, 15*time.Minute),
		AccountLimiter: auth.NewLimiter(30, 15*time.Minute),
		KubeLimiter:    limiter,
		Opts:           cfg.opts,
		Log:            log.With("component", "api"),
		Cluster:        clusterSummary(kc, restCfg, cfg),
		ClusterService: newClusterService(mgr, kc, restCfg, svc.settings, upgradeCfg.Enabled(), cfg, log),
		Trigger:        rec.Trigger,
		Ops:            svc.ops,
		Schedules:      svc.schedules,
		DevMode:        devMode,
		SetupToken:     boot.setupToken,
		Version:        appVersion,
		UpgradeChecker: updates,
		Upgrader:       upgrader,
	}
	return &panel{manager: mgr, api: api}, nil
}

// newClusterService gathers the node, hardware, identity and health details of the cluster page.
func newClusterService(
	mgr ctrl.Manager, kc *kube.Client, restCfg *rest.Config, store *settings.Store, selfUpgrade bool, cfg config,
	log *slog.Logger,
) *clusterinfo.Service {
	return &clusterinfo.Service{
		Kube: kc, Config: restCfg, Namespace: cfg.opts.Namespace, HelperImage: cfg.opts.HelperImage,
		KubeContext: cfg.kubeContext, HTTP1: cfg.http1, Log: log.With("component", "clusterinfo"),
		Reader: mgr.GetAPIReader(), Settings: store, SelfUpgrade: selfUpgrade, AdmissionPolicy: cfg.admissionPolicy,
	}
}

// services are the domain services shared by the API, the controllers and the background jobs.
type services struct {
	files     *files.Service
	ops       *serverctl.Ops
	schedules *schedule.Runner
	settings  *settings.Store
	eggs      *eggstore.Store
}

// setupServices builds the domain services and registers their background runnables (schedules,
// egg auto updates) with the manager.
func setupServices(
	ctx context.Context, mgr ctrl.Manager, kc *kube.Client, hub *console.Hub, activity *files.Activity,
	rec *controller.Reconciler, limiter *kube.RateLimiter, cfg config, log *slog.Logger,
) (*services, error) {
	ns := cfg.opts.Namespace
	filesService := &files.Service{
		Manager: &files.Manager{Kube: kc}, Activity: activity, Reader: mgr.GetAPIReader(), Trigger: rec.Trigger,
	}
	ops := &serverctl.Ops{
		Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Kube: kc, Hub: hub, Namespace: ns, Files: filesService,
	}
	schedules := &schedule.Runner{
		Client:   mgr.GetClient(),
		Reader:   mgr.GetAPIReader(),
		Ops:      ops,
		Files:    filesService,
		Hub:      hub,
		Trigger:  rec.Trigger,
		Location: schedule.Location(cfg.opts.Timezone),
		Log:      log.With("component", "schedules"),
	}
	if err := mgr.Add(schedules); err != nil {
		return nil, fmt.Errorf("setting up schedules: %w", err)
	}
	// The controller runs the "Tasks" (event schedules) when a server started and before it stops.
	rec.Events = schedules
	eggs := &eggstore.Store{Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: ns}
	// Eggs with "update automatically" are checked against their update URL every hour.
	updater := &eggstore.Updater{Store: eggs, Interval: time.Hour, Log: log.With("component", "eggs")}
	if err := mgr.Add(updater); err != nil {
		return nil, fmt.Errorf("setting up egg auto updates: %w", err)
	}
	// The stored Kubernetes API limit applies from the start and after every save.
	store := &settings.Store{Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: ns, Limiter: limiter}
	set, err := store.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading panel settings: %w", err)
	}
	limiter.SetLimits(int(set.KubeAPIQPS), int(set.KubeAPIUserQPS))
	if err := store.Watch(ctx, mgr.GetCache()); err != nil {
		return nil, fmt.Errorf("watching panel settings: %w", err)
	}
	return &services{files: filesService, ops: ops, schedules: schedules, settings: store, eggs: eggs}, nil
}

// newManager creates the controller-runtime manager (shared cache, no metrics/health server).
func newManager(ctx context.Context, restCfg *rest.Config, scheme *runtime.Scheme) (ctrl.Manager, error) {
	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme:                 scheme,
		Cache:                  bootstrap.CacheOptions(),
		Client:                 client.Options{Cache: &client.CacheOptions{DisableFor: bootstrap.UncachedTypes()}},
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		return nil, fmt.Errorf("creating manager: %w", err)
	}
	if err := mgr.GetFieldIndexer().
		IndexField(ctx, &v1alpha1.GameServer{}, httpapi.IndexServerName, httpapi.ServerNameIndex); err != nil {
		return nil, fmt.Errorf("indexing game servers: %w", err)
	}
	return mgr, nil
}

// setupControllers registers the game server and user controllers and the files pod reaper.
func setupControllers(
	mgr ctrl.Manager, kc *kube.Client, hub *console.Hub, activity *files.Activity, cfg config, log *slog.Logger,
) (*controller.Reconciler, error) {
	rec := &controller.Reconciler{
		Client: mgr.GetClient(),
		Reader: mgr.GetAPIReader(),
		Kube:   kc,
		Hub:    hub,
		Opts:   cfg.opts,
		Files:  activity,
		Pools:  &settings.PoolResolver{Reader: mgr.GetAPIReader()},
		Log:    log.With("component", "controller"),
	}
	if err := rec.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("setting up controller: %w", err)
	}
	// The "done" line of a game pod marks the server as running: reconcile right away.
	hub.OnDone = rec.Trigger
	if err := mgr.Add(
		&controller.FilesReaper{
			Client: mgr.GetClient(),
			Reader: mgr.GetAPIReader(),
			Kube:   kc,
			Files:  activity,
			Log:    log.With("component", "files"),
		},
	); err != nil {
		return nil, fmt.Errorf("setting up files reaper: %w", err)
	}
	users := &controller.UserReconciler{
		Client: mgr.GetClient(),
		Reader: mgr.GetAPIReader(),
		Log:    log.With("component", "users"),
	}
	if err := users.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("setting up user controller: %w", err)
	}
	return rec, nil
}

// setupSelfUpgrade enables update checks and upgrade jobs when the chart set the environment
// variables (chart value selfUpgrade.enabled); otherwise checker and upgrader are nil.
func setupSelfUpgrade(
	mgr ctrl.Manager, kc *kube.Client, c selfupgrade.Config, namespace string, log *slog.Logger,
) (selfupgrade.Config, *selfupgrade.Checker, *selfupgrade.Upgrader, error) {
	c.Namespace = namespace
	if !c.Enabled() {
		return c, nil, nil, nil
	}
	updates := &selfupgrade.Checker{Config: c, Log: log.With("component", "upgrade")}
	if err := mgr.Add(updates); err != nil {
		return c, nil, nil, fmt.Errorf("setting up update checks: %w", err)
	}
	upgrader := &selfupgrade.Upgrader{Config: c, Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Kube: kc}
	if err := mgr.Add(upgrader.PruneEvery(time.Hour, log.With("component", "upgrade"))); err != nil {
		return c, nil, nil, fmt.Errorf("setting up upgrade job cleanup: %w", err)
	}
	return c, updates, upgrader, nil
}

// clusterSummary is the connection summary of the sidebar cluster card.
func clusterSummary(kc *kube.Client, restCfg *rest.Config, cfg config) httpapi.ClusterInfo {
	info := httpapi.ClusterInfo{
		Context:          currentContext(cfg.kubeContext),
		Server:           restCfg.Host,
		Namespace:        cfg.opts.Namespace,
		StorageClass:     cfg.opts.StorageClass,
		LoadBalancerPool: cfg.opts.LoadBalancerPool,
	}
	if v, err := kc.Clientset.Discovery().ServerVersion(); err == nil {
		info.Version = v.GitVersion
	}
	return info
}
