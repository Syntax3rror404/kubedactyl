package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"app/api/v1alpha1"
	"app/base/api"
	"app/base/auth"
	"app/base/bootstrap"
	"app/base/httpserver"
	"app/base/kube"
	"app/base/selfupgrade"
	"app/base/settings"
	"app/base/users"
	"app/config/crd"
	"app/internal/appsettings"
	"app/internal/appusers"
	"app/internal/clusterinfo"
	"app/internal/console"
	"app/internal/controller"
	"app/internal/diagnostics"
	"app/internal/eggstore"
	"app/internal/files"
	"app/internal/httpapi"
	"app/internal/schedule"
	"app/internal/serverctl"
	"app/internal/tenancy"
)

// prepareCluster runs the start-up steps with a direct client (the cache does not run yet): the base steps (CRDs,
// namespace, session key, first admin), then the user namespaces of this installation and the settings, created
// with the storage class and pool of the flags on the first start.
func prepareCluster(
	ctx context.Context, restCfg *rest.Config, scheme *runtime.Scheme, cfg config, log *slog.Logger,
) (*bootstrap.Result, error) {
	ns := cfg.Namespace
	direct, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	boot, err := bootstrap.Run(
		ctx, direct, crd.Files, ns, httpserver.LocalURL(cfg.Listen), log.With("component", "bootstrap"),
	)
	if err != nil {
		return nil, err
	}
	// The panel namespace also scopes this installation to its user namespaces. In the cluster the panel may
	// only write in user namespaces through its tenant role.
	tenancy.SystemNamespace = ns
	tenancy.NamespacePrefix = ns + "-user-"
	tenancy.TenantRole, tenancy.PanelServiceAccount = cfg.tenantRole, cfg.serviceAccount
	if err := appsettings.Bootstrap(
		ctx, direct, ns, cfg.opts.StorageClass, cfg.opts.LoadBalancerPool, log.With("component", "settings"),
	); err != nil {
		return nil, fmt.Errorf("creating panel settings: %w", err)
	}
	return boot, nil
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
	boot *bootstrap.Result, devMode bool, log *slog.Logger,
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
	updates, upgrader, err := selfupgrade.Setup(mgr, kc, cfg.Namespace, appVersion, log)
	if err != nil {
		return nil, err
	}
	svc, err := setupServices(ctx, mgr, kc, hub, activity, rec, limiter, cfg, log)
	if err != nil {
		return nil, err
	}
	clusterService := newClusterService(mgr, kc, restCfg, svc.settings, upgrader != nil, cfg, log)
	// The health checks the enabled pools and storage classes: a settings change shows at once.
	svc.settings.Changed = clusterService.ForgetHealth

	base := &api.API{
		Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: cfg.Namespace,
		Log: log.With("component", "api"), Settings: svc.settings, Users: svc.users, Signer: boot.Signer,
		// Failed sign-ins: per client and user, per client (password spraying), per user.
		Limiter:        auth.NewLimiter(5, 5*time.Minute),
		ClientLimiter:  auth.NewLimiter(20, 15*time.Minute),
		AccountLimiter: auth.NewLimiter(30, 15*time.Minute),
		Proxies:        cfg.proxies,
		KubeLimiter:    limiter,
		DevMode:        devMode,
		SetupToken:     boot.SetupToken,
		Version:        appVersion,
		UpgradeChecker: updates,
		Upgrader:       upgrader,
	}
	handlers := &httpapi.API{
		API:            base,
		Kube:           kc,
		Hub:            hub,
		Files:          svc.files,
		Eggs:           svc.eggs,
		Library:        eggstore.NewLibrary(),
		Diagnostics:    &diagnostics.Diagnoser{},
		Opts:           cfg.opts,
		Cluster:        clusterSummary(kc, restCfg, cfg),
		ClusterService: clusterService,
		Trigger:        rec.Trigger,
		Ops:            svc.ops,
		Schedules:      svc.schedules,
	}
	return &panel{manager: mgr, api: handlers}, nil
}

// newClusterService gathers the node, hardware, identity and health details of the cluster page.
func newClusterService(
	mgr ctrl.Manager, kc *kube.Client, restCfg *rest.Config, store *settings.Store, selfUpgrade bool, cfg config,
	log *slog.Logger,
) *clusterinfo.Service {
	return &clusterinfo.Service{
		Kube: kc, Config: restCfg, Namespace: cfg.Namespace, HelperImage: cfg.opts.HelperImage,
		KubeContext: cfg.KubeContext, HTTP1: cfg.HTTP1, Log: log.With("component", "clusterinfo"),
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
	users     *users.Store
}

// setupServices builds the domain services and registers their background runnables (schedules,
// egg auto updates) with the manager.
func setupServices(
	ctx context.Context, mgr ctrl.Manager, kc *kube.Client, hub *console.Hub, activity *files.Activity,
	rec *controller.Reconciler, limiter *kube.RateLimiter, cfg config, log *slog.Logger,
) (*services, error) {
	ns := cfg.Namespace
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
	// The controller runs the "Tasks" (event schedules) when a server started and before it stops,
	// and keeps the files pod while the file service uses it.
	rec.Events, rec.Files = schedules, filesService
	eggs := &eggstore.Store{Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: ns}
	// Eggs with "update automatically" are checked against their update URL every hour.
	updater := &eggstore.Updater{Store: eggs, Interval: time.Hour, Log: log.With("component", "eggs")}
	if err := mgr.Add(updater); err != nil {
		return nil, fmt.Errorf("setting up egg auto updates: %w", err)
	}
	// The stored Kubernetes API limit applies from the start and after every save, and changes made with kubectl
	// reach the store through the informer.
	store := &settings.Store{
		Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: ns, Limiter: limiter,
		App: appsettings.Settings{Reader: mgr.GetAPIReader()},
	}
	if err := store.Start(ctx, mgr.GetCache()); err != nil {
		return nil, err
	}
	accounts := &users.Store{
		Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: ns,
		App: &appusers.Accounts{Client: mgr.GetClient(), Reader: mgr.GetAPIReader(), Namespace: ns, Settings: store},
	}
	return &services{
		files: filesService, ops: ops, schedules: schedules, settings: store, eggs: eggs, users: accounts,
	}, nil
}

// newManager creates the controller-runtime manager (shared cache, no metrics/health server).
func newManager(ctx context.Context, restCfg *rest.Config, scheme *runtime.Scheme) (ctrl.Manager, error) {
	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme:                 scheme,
		Cache:                  cacheOptions(),
		Client:                 client.Options{Cache: &client.CacheOptions{DisableFor: uncachedTypes()}},
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

// cacheOptions caches only the objects the panel created, so it never keeps all pods or services of the cluster
// in memory.
func cacheOptions() cache.Options {
	managed := labels.SelectorFromSet(labels.Set{tenancy.LabelManagedBy: tenancy.ManagedBy})
	return cache.Options{ByObject: map[client.Object]cache.ByObject{
		&corev1.Pod{}:                   {Label: managed},
		&corev1.Service{}:               {Label: managed},
		&corev1.PersistentVolumeClaim{}: {Label: managed},
		&corev1.ConfigMap{}:             {Label: managed},
		&networkingv1.NetworkPolicy{}:   {Label: managed},
	}}
}

// uncachedTypes are always read from the API server: secrets and the cluster wide objects would otherwise be
// cached for the whole cluster.
func uncachedTypes() []client.Object {
	return []client.Object{
		&corev1.Secret{},
		&corev1.Namespace{},
		&corev1.PersistentVolume{},
		&apiextensionsv1.CustomResourceDefinition{},
		&rbacv1.RoleBinding{},
	}
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
		Pools:  &appsettings.PoolResolver{Reader: mgr.GetAPIReader()},
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
		Client:    mgr.GetClient(),
		Reader:    mgr.GetAPIReader(),
		Namespace: cfg.Namespace,
		Log:       log.With("component", "users"),
	}
	if err := users.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("setting up user controller: %w", err)
	}
	return rec, nil
}

// clusterSummary is the connection summary of the sidebar cluster card.
func clusterSummary(kc *kube.Client, restCfg *rest.Config, cfg config) httpapi.ClusterInfo {
	info := httpapi.ClusterInfo{
		Context:          kube.CurrentContext(cfg.KubeContext),
		Server:           restCfg.Host,
		Namespace:        cfg.Namespace,
		StorageClass:     cfg.opts.StorageClass,
		LoadBalancerPool: cfg.opts.LoadBalancerPool,
	}
	if v, err := kc.Clientset.Discovery().ServerVersion(); err == nil {
		info.Version = v.GitVersion
	}
	return info
}
