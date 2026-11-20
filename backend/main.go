// Command kubedactyl is the panel: one binary with the web interface, the REST/websocket API and
// the Kubernetes controllers. `kubedactyl upgrade …` runs the self-upgrade job instead.
// main.go only parses the configuration and starts the parts that wiring.go puts together.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	// Time zone database for schedules and TZ, also in images without tzdata (Alpine).
	_ "time/tzdata"

	"app/app"
	"app/base/auth"
	"app/base/bootstrap"
	"app/base/cli"
	"app/base/httpserver"
	"app/base/kube"
	"app/base/selfupgrade"
	"app/base/settings"
	"app/docs" // generated with: go tool swag init
	"app/internal/gameserver"
	"app/web"
)

// appVersion is set at build time from the VERSION file: -ldflags "-X main.appVersion=…".
var appVersion = "dev"

// config is the panel configuration from flags and environment variables.
type config struct {
	cli.Flags
	opts gameserver.Options

	// From the environment only (set by the chart): tenantRole, serviceAccount and admissionPolicy describe
	// the panel's permissions in the cluster (tenant role bound per user namespace, admission policy that
	// limits them).
	tenantRole, serviceAccount, admissionPolicy string
	// proxies are the parsed --trusted-proxies.
	proxies auth.Proxies
}

// @title						Kubedactyl API
// @version					dev
// @description				Game server panel for Kubernetes that runs Pterodactyl and Pelican eggs.
// @description				Request bodies are strict JSON: unknown fields are refused (400).
// @BasePath					/api
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				"Bearer <token>": a session token from /auth/login or an API token (kdt_…). The web UI
// @description				uses the kd_session cookie.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// `kubedactyl upgrade …` runs inside the self-upgrade job.
	if len(os.Args) > 1 && os.Args[1] == "upgrade" {
		log := cli.SetupLogging(false)
		if err := selfupgrade.Main(ctx, os.Args[2:], appVersion, log); err != nil {
			cli.Fatal(log, "upgrade failed", err)
		}
		return
	}
	cfg := parseFlags()
	cli.PrintBanner(os.Stderr, appVersion, cli.ColorTerminal(os.Stderr)) // before the log lines, on the same stream
	// The API reference shows the version of this binary, not the one at generation time.
	docs.SwaggerInfo.Version = appVersion
	log := cli.SetupLogging(cfg.Verbose)
	if err := run(ctx, cfg, log); err != nil {
		cli.Fatal(log, "panel stopped", err)
	}
}

// run starts the panel: prepare the cluster (CRDs, namespace, first admin, settings), build
// controllers, services and API (wiring.go), then serve HTTP until ctx ends.
func run(ctx context.Context, cfg config, log *slog.Logger) error {
	proxies, err := auth.ParseProxies(cfg.TrustedProxies)
	if err != nil {
		return err
	}
	cfg.proxies = proxies
	limiter := kube.NewRateLimiter(settings.DefaultKubeAPIQPS, settings.DefaultKubeAPIUserQPS)
	restCfg, err := kube.RestConfig(cfg.KubeContext, cfg.HTTP1, limiter)
	if err != nil {
		return err
	}
	scheme, err := bootstrap.Scheme()
	if err != nil {
		return err
	}
	boot, err := prepareCluster(ctx, restCfg, scheme, cfg, log)
	if err != nil {
		return err
	}
	frontend := web.FS() // nil in development mode (Vite serves the UI)
	p, err := newPanel(ctx, restCfg, limiter, scheme, cfg, boot, frontend == nil, log)
	if err != nil {
		return err
	}
	router, err := httpserver.NewRouter(httpserver.Config{
		API: p.api, Settings: p.api.Settings, Frontend: frontend, Version: appVersion, Log: log,
	})
	if err != nil {
		return err
	}
	go func() {
		if err := p.manager.Start(ctx); err != nil {
			cli.Fatal(log, "controller manager stopped", err)
		}
	}()
	log.Info("panel started", "name", app.Name, "version", appVersion, "url", httpserver.LocalURL(cfg.Listen),
		"context", p.api.Cluster.Context, "namespace", cfg.Namespace)
	return httpserver.Serve(ctx, cfg.Listen, router)
}

func parseFlags() config {
	var cfg config
	cfg.Bind("panel namespace (eggs, users, settings); game servers run in <namespace>-user-<user>")
	o := &cfg.opts
	flag.StringVar(
		&o.StorageClass, "storage-class", cli.EnvOr("STORAGE_CLASS", "longhorn"),
		"storage class enabled in the panel settings on the first start",
	)
	flag.StringVar(
		&o.LoadBalancerPool, "lb-pool", cli.EnvOr("LB_POOL", "general"),
		"Cilium LB IPAM pool enabled in the panel settings on the first start (name or lb.cilium.io/pool label value)",
	)
	// Same tag as the image base (Dockerfile, scripts/imagebuild) and the chart's panel.helperImage.
	flag.StringVar(
		&o.HelperImage, "helper-image", cli.EnvOr("HELPER_IMAGE", "alpine:3.24.4"),
		"image of the file helper and hardware probe pods",
	)
	flag.StringVar(
		&o.DockerInterface, "docker-interface", cli.EnvOr("DOCKER_INTERFACE", "127.0.0.1"),
		"value of {{config.docker.interface}} in egg config files",
	)
	flag.StringVar(&o.Timezone, "timezone", cli.EnvOr("TZ", "UTC"), "TZ passed to game servers")
	flag.Parse()
	o.Namespace = cfg.Namespace
	cfg.tenantRole = os.Getenv("KUBEDACTYL_TENANT_ROLE")
	cfg.serviceAccount = os.Getenv("KUBEDACTYL_SERVICE_ACCOUNT")
	cfg.admissionPolicy = os.Getenv("KUBEDACTYL_ADMISSION_POLICY")
	return cfg
}
