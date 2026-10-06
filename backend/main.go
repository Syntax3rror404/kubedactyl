// Command kubedactyl is the panel: one binary with the web interface, the REST/websocket API and
// the Kubernetes controllers. `kubedactyl upgrade …` runs the self-upgrade job instead.
// main.go only parses the configuration and starts the parts that wiring.go puts together.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Time zone database for schedules and TZ, also in images without tzdata (Alpine).
	_ "time/tzdata"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"app/docs" // generated with: go tool swag init
	"app/internal/gameserver"
	"app/internal/httpserver"
	"app/internal/kube"
	"app/internal/selfupgrade"
	"app/internal/settings"
	"app/web"
)

const appName = "Kubedactyl"

// appVersion is set at build time from the VERSION file: -ldflags "-X main.appVersion=…".
var appVersion = "dev"

// config is the panel configuration from flags and environment variables.
type config struct {
	kubeContext string
	addr        string
	verbose     bool
	proxies     string
	opts        gameserver.Options

	// From the environment only (set by the chart):
	// http1 forces HTTP/1.1 to the API server (unless KUBE_HTTP2 is set).
	http1 bool
	// tenantRole, serviceAccount and admissionPolicy describe the panel's permissions in the
	// cluster (tenant role bound per user namespace, admission policy that limits them).
	tenantRole, serviceAccount, admissionPolicy string
	// upgrade configures self-upgrades (enabled when every field is set).
	upgrade selfupgrade.Config
}

// @title						Kubedactyl API
// @version					dev
// @description				Game server panel for Kubernetes that runs Pterodactyl and Pelican eggs.
// @BasePath					/api
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				"Bearer <token>": a session token from /auth/login or an API token (kdt_…). The web UI
// @description				uses the kd_session cookie.
func main() {
	// `kubedactyl upgrade …` runs inside the self-upgrade job.
	if len(os.Args) > 1 && os.Args[1] == "upgrade" {
		runUpgrade(os.Args[2:])
		return
	}
	cfg := parseFlags()
	printBanner(os.Stderr, appVersion, colorTerminal(os.Stderr)) // before the log lines, on the same stream
	// The Swagger UI shows the version of this binary, not the one at generation time.
	docs.SwaggerInfo.Version = appVersion
	log := setupLogging(cfg.verbose)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, log); err != nil {
		fatal(log, "panel stopped", err)
	}
}

// run starts the panel: prepare the cluster (CRDs, namespace, first admin, settings), build
// controllers, services and API (wiring.go), then serve HTTP until ctx ends.
func run(ctx context.Context, cfg config, log *slog.Logger) error {
	limiter := kube.NewRateLimiter(settings.DefaultKubeAPIQPS, settings.DefaultKubeAPIUserQPS)
	restCfg, err := kubeConfig(cfg.kubeContext, cfg.http1, limiter)
	if err != nil {
		return err
	}
	scheme, err := newScheme()
	if err != nil {
		return err
	}
	boot, err := prepareCluster(ctx, restCfg, scheme, cfg, log)
	if err != nil {
		return err
	}
	frontend := web.FS() // nil in development mode (Vite serves the UI)
	if frontend != nil {
		gin.SetMode(gin.ReleaseMode)
	}
	p, err := newPanel(ctx, restCfg, limiter, scheme, cfg, boot, frontend == nil, log)
	if err != nil {
		return err
	}
	router, err := httpserver.NewRouter(httpserver.Config{
		API: p.api, Frontend: frontend, TrustedProxies: httpserver.ParseTrustedProxies(cfg.proxies),
		Name: appName, Version: appVersion, Log: log,
	})
	if err != nil {
		return err
	}
	go func() {
		if err := p.manager.Start(ctx); err != nil {
			fatal(log, "controller manager stopped", err)
		}
	}()
	log.Info("panel started", "name", appName, "version", appVersion, "url", "http://localhost"+cfg.addr,
		"context", p.api.Cluster.Context, "namespace", cfg.opts.Namespace)
	return serve(ctx, cfg.addr, router)
}

func parseFlags() config {
	var cfg config
	flag.StringVar(
		&cfg.kubeContext, "kube-context", os.Getenv("KUBE_CONTEXT"),
		"kubeconfig context (default: current context, or in-cluster config)",
	)
	flag.StringVar(&cfg.addr, "listen", ":"+envOr("PORT", "8080"), "HTTP listen address")
	flag.BoolVar(&cfg.verbose, "verbose", os.Getenv("VERBOSE") != "", "enable debug logging")
	flag.StringVar(
		&cfg.proxies, "trusted-proxies", os.Getenv("TRUSTED_PROXIES"),
		"comma separated IPs/CIDRs of reverse proxies whose X-Forwarded-For is trusted",
	)
	o := &cfg.opts
	flag.StringVar(
		&o.Namespace, "namespace", envOr("KUBEDACTYL_NAMESPACE", "kubedactyl"),
		"panel namespace (eggs, users, settings); game servers run in <namespace>-user-<user>",
	)
	flag.StringVar(
		&o.StorageClass, "storage-class", envOr("STORAGE_CLASS", "longhorn"),
		"storage class enabled in the panel settings on the first start",
	)
	flag.StringVar(
		&o.LoadBalancerPool,
		"lb-pool",
		envOr("LB_POOL", "general"),
		"Cilium LB IPAM pool enabled in the panel settings on the first start (name or lb.cilium.io/pool label value)",
	)
	flag.StringVar(
		&o.HelperImage, "helper-image", envOr("HELPER_IMAGE", "alpine:3.24.2"),
		"image of the file helper and hardware probe pods",
	)
	flag.StringVar(
		&o.DockerInterface, "docker-interface", envOr("DOCKER_INTERFACE", "127.0.0.1"),
		"value of {{config.docker.interface}} in egg config files",
	)
	flag.StringVar(&o.Timezone, "timezone", envOr("TZ", "UTC"), "TZ passed to game servers")
	flag.Parse()
	cfg.http1 = os.Getenv("KUBE_HTTP2") == ""
	cfg.tenantRole = os.Getenv("KUBEDACTYL_TENANT_ROLE")
	cfg.serviceAccount = os.Getenv("KUBEDACTYL_SERVICE_ACCOUNT")
	cfg.admissionPolicy = os.Getenv("KUBEDACTYL_ADMISSION_POLICY")
	cfg.upgrade = selfupgrade.Config{
		Chart:   os.Getenv("KUBEDACTYL_UPGRADE_CHART"),
		Release: os.Getenv("KUBEDACTYL_HELM_RELEASE"),
		ServiceAccount: os.Getenv(
			"KUBEDACTYL_UPGRADE_SERVICE_ACCOUNT",
		),
		Image:   os.Getenv("KUBEDACTYL_IMAGE"),
		Current: appVersion,
	}
	return cfg
}

// setupLogging routes slog, controller-runtime and klog through one handler.
func setupLogging(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	log := slog.New(quietHandler{slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})})
	slog.SetDefault(log)
	logger := logr.FromSlogHandler(log.Handler())
	ctrl.SetLogger(logger)
	klog.SetLogger(logger)
	return log
}

// serve runs the HTTP server until ctx ends and then shuts it down gracefully.
func serve(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// runUpgrade is `kubedactyl upgrade`: helm upgrade of the panel release (see selfupgrade.Run).
func runUpgrade(args []string) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	opts, err := selfupgrade.ParseArgs(args)
	if err != nil {
		fatal(log, "upgrade arguments", err)
	}
	klog.SetLogger(logr.FromSlogHandler(log.Handler()))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Info(
		"upgrading the panel", "release", opts.Release, "namespace", opts.Namespace, "chart", opts.Chart, "version",
		opts.Version, "from", appVersion,
	)
	if err := selfupgrade.Run(ctx, opts, log); err != nil {
		fatal(log, "upgrade failed", err)
	}
}

func fatal(log *slog.Logger, msg string, err error) {
	log.Error(msg, "err", err)
	os.Exit(1)
}

// currentContext names the kubeconfig context in use (for the cluster page).
func currentContext(explicit string) string {
	if explicit != "" {
		return explicit
	}
	raw, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil || raw.CurrentContext == "" {
		return "in-cluster"
	}
	return raw.CurrentContext
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
