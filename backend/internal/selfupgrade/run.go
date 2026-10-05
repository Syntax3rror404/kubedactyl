package selfupgrade

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v4/pkg/action"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/registry"
	releasev1 "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// RunOptions are the arguments of `kubedactyl upgrade`.
type RunOptions struct {
	Release, Namespace, Chart, Version string
	// ChartFile is a packaged chart (.tgz) used instead of pulling Chart (offline upgrades).
	ChartFile string
	// KubeContext selects a kubeconfig context; empty means in-cluster (the job).
	KubeContext string
}

// ParseArgs reads the arguments of `kubedactyl upgrade`.
func ParseArgs(args []string) (RunOptions, error) {
	var o RunOptions
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.StringVar(&o.Release, "release", "", "Helm release name")
	fs.StringVar(&o.Namespace, "release-namespace", "", "namespace of the release")
	fs.StringVar(&o.Chart, "chart", "", "OCI chart reference without tag (oci://…)")
	fs.StringVar(&o.Version, "version", "", "chart version to upgrade to")
	fs.StringVar(
		&o.ChartFile, "chart-file", "",
		"packaged chart (.tgz) instead of pulling --chart (its version must match --version)",
	)
	fs.StringVar(&o.KubeContext, "kube-context", "", "kubeconfig context (default: in-cluster)")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if o.Release == "" || o.Namespace == "" || (o.Chart == "") == (o.ChartFile == "") || o.Version == "" {
		return o, errors.New(
			"--release, --release-namespace, --version and either --chart or --chart-file are required",
		)
	}
	if _, err := semver.StrictNewVersion(o.Version); err != nil {
		return o, fmt.Errorf("--version: %w", err)
	}
	return o, nil
}

// Run upgrades the release to the chart version like
// `helm upgrade <release> <chart> --version <v> --reset-then-reuse-values`: the new chart's
// defaults with the values the release was installed or last upgraded with.
func Run(ctx context.Context, o RunOptions, log *slog.Logger) error {
	// Same field manager as the helm CLI (the default is the binary name): server-side apply
	// would otherwise report conflicts with the fields a manual helm install/upgrade owns.
	kube.ManagedFieldsManager = "helm"
	flags := genericclioptions.NewConfigFlags(false)
	flags.Namespace = &o.Namespace
	if o.KubeContext != "" {
		flags.Context = &o.KubeContext
	}
	cfg := action.NewConfiguration()
	cfg.SetLogger(log.Handler())
	if err := cfg.Init(flags, o.Namespace, "secret"); err != nil {
		return fmt.Errorf("helm: %w", err)
	}
	cur, err := action.NewGet(cfg).Run(o.Release)
	if err != nil {
		return fmt.Errorf("reading release %s: %w", o.Release, err)
	}
	rel, ok := cur.(*releasev1.Release)
	if !ok {
		return fmt.Errorf("release %s has an unsupported format %T", o.Release, cur)
	}
	log.Info(
		"current release", "release", rel.Name, "revision", rel.Version, "chart", rel.Chart.Metadata.Version, "status",
		rel.Info.Status,
	)

	rc, err := newRegistryClient()
	if err != nil {
		return err
	}
	ch, src, err := loadChart(rc, o)
	if err != nil {
		return err
	}
	if ch.Metadata.Version != o.Version {
		return fmt.Errorf("%s has version %s, not %s", src, ch.Metadata.Version, o.Version)
	}
	if ch.Metadata.Name != rel.Chart.Metadata.Name {
		return fmt.Errorf(
			"%s contains the chart %q, the release uses %q", src, ch.Metadata.Name, rel.Chart.Metadata.Name,
		)
	}

	up := action.NewUpgrade(cfg)
	up.Namespace = o.Namespace
	up.SetRegistryClient(rc)
	up.ResetThenReuseValues = true
	up.WaitStrategy = kube.HookOnlyStrategy
	up.Timeout = 5 * time.Minute
	up.MaxHistory = 10
	res, err := up.RunWithContext(ctx, o.Release, ch, map[string]any{})
	if err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}
	if r, ok := res.(*releasev1.Release); ok {
		log.Info(
			"upgraded", "release", r.Name, "revision", r.Version, "chart", r.Chart.Metadata.Version, "status",
			r.Info.Status,
		)
	}
	return nil
}

// loadChart pulls the chart version from the registry, or loads --chart-file.
func loadChart(rc *registry.Client, o RunOptions) (*chartv2.Chart, string, error) {
	if o.ChartFile != "" {
		ch, err := loader.Load(o.ChartFile)
		if err != nil {
			return nil, o.ChartFile, fmt.Errorf("loading %s: %w", o.ChartFile, err)
		}
		return ch, o.ChartFile, nil
	}
	ref := Config{Chart: o.Chart}.registryRef() + ":" + o.Version
	pulled, err := rc.Pull(ref)
	if err != nil {
		return nil, ref, fmt.Errorf("pulling %s: %w", ref, err)
	}
	ch, err := loader.LoadArchive(bytes.NewReader(pulled.Chart.Data))
	if err != nil {
		return nil, ref, fmt.Errorf("loading %s: %w", ref, err)
	}
	return ch, ref, nil
}
