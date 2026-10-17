package gameserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"app/api/v1alpha1"
	"app/internal/configfile"
	"app/internal/kube"
)

// readConfigScript prints at most "$2" bytes of the config file "$1" (exit 3: missing or outside
// the volume).
const readConfigScript = `[ -f "$1" ] || exit 3
inside "$(real "$1")"
head -c "$2" "$1"`

// PreStart runs the steps before every start: it updates the egg config files as the game server
// user in the files pod (whose init container fixed the file ownership already). It returns
// warnings for files that could not be parsed or are too large.
func PreStart(
	ctx context.Context, k *kube.Client, gs *v1alpha1.GameServer, e *v1alpha1.Egg, opts Options,
) ([]string, error) {
	env := Environment(gs, e, opts)
	cfg := configfile.NewConfig(opts.DockerInterface)
	var warnings []string
	for _, cf := range e.Spec.ConfigFiles {
		warning, err := updateConfigFile(ctx, k, gs, cf, env, cfg)
		if err != nil {
			return warnings, err
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return warnings, nil
}

// updateConfigFile applies the replacements of one egg config file: a missing file is created
// when its parser can, a file larger than configfile.MaxFileSize is skipped.
func updateConfigFile(
	ctx context.Context, k *kube.Client, gs *v1alpha1.GameServer, cf v1alpha1.ConfigFile,
	env map[string]string, cfg configfile.Config,
) (warning string, err error) {
	pod := FilesPodName(gs.Name)
	abs := path.Join(ServerRoot, path.Clean("/"+cf.File))
	// One byte more than the limit tells a file that is too large.
	content := kube.LimitedBuffer{Max: configfile.MaxFileSize + 1}
	exists := true
	err = k.Exec(ctx, gs.Namespace, pod, FilesContainerName,
		[]string{"sh", "-c", PathScript + readConfigScript, "sh", abs, strconv.Itoa(content.Max)}, nil, &content)
	var exitErr *kube.ExitError
	switch {
	case errors.As(err, &exitErr) && exitErr.Code == 3:
		exists = false
	case content.Len() > configfile.MaxFileSize:
		return fmt.Sprintf("Skipped %s: larger than %d bytes", cf.File, configfile.MaxFileSize), nil
	case err != nil:
		return "", fmt.Errorf("read %s: %w", cf.File, err)
	}

	reps := make([]configfile.Replacement, 0, len(cf.Replace))
	for _, r := range cf.Replace {
		valueType := r.ValueType
		if strings.TrimSpace(valueType) == "" {
			valueType = "string"
		}
		reps = append(reps, configfile.Replacement{
			Match: r.Match, IfValue: r.IfValue, Type: configfile.ValueType(valueType),
			Value: ResolvePlaceholders(r.ReplaceWith, gs, env),
		})
	}
	out, write, err := configfile.Apply(cf.Parser, content.Bytes(), exists, reps, cfg)
	if err != nil {
		return fmt.Sprintf("Failed to update %s (%s parser): %v", cf.File, cf.Parser, err), nil
	}
	if !write {
		return "", nil
	}
	err = k.Exec(ctx, gs.Namespace, pod, FilesContainerName,
		[]string{"sh", "-c", PathScript + WriteFileScript, "sh", abs}, bytes.NewReader(out), nil)
	if err != nil {
		return "", fmt.Errorf("write %s: %w", cf.File, err)
	}
	return "", nil
}

// DiskUsage measures the server files with du inside a pod that mounts the data volume.
func DiskUsage(ctx context.Context, kc *kube.Client, namespace, pod, container string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// du exits with 1 when a directory is unreadable but still prints the total.
	out, err := kc.ExecOutput(
		ctx, namespace, pod, container, []string{"sh", "-c", "du -sk " + ServerRoot + " 2>/dev/null; true"}, nil,
	)
	if err != nil {
		return 0, err
	}
	kb, err := strconv.ParseInt(strings.Fields(string(out) + " x")[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return kb * 1024, nil
}
