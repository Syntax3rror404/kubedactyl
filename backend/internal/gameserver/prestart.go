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

// PreStart runs the steps before every start: update the egg config files (as the game
// server user in the files pod, whose init container already fixed the file ownership).
// It returns warnings for files that could not be parsed.
func PreStart(
	ctx context.Context, k *kube.Client, gs *v1alpha1.GameServer, e *v1alpha1.Egg, opts Options,
) ([]string, error) {
	pod := FilesPodName(gs.Name)
	env := Environment(gs, e, opts)
	cfg := configfile.NewConfig(opts.DockerInterface)
	var warnings []string

	for _, cf := range e.Spec.ConfigFiles {
		abs := path.Join(ServerRoot, path.Clean("/"+cf.File))
		// One byte more than the limit tells a file that is too large.
		content := kube.LimitedBuffer{Max: configfile.MaxFileSize + 1}
		exists := true
		err := k.Exec(ctx, gs.Namespace, pod, FilesContainerName,
			[]string{"sh", "-c", PathScript + readConfigScript, "sh", abs, strconv.Itoa(content.Max)}, nil, &content)
		var exitErr *kube.ExitError
		switch {
		case errors.As(err, &exitErr) && exitErr.Code == 3:
			exists = false
		case content.Len() > configfile.MaxFileSize:
			warnings = append(
				warnings,
				fmt.Sprintf("Skipped %s: larger than %d bytes", cf.File, configfile.MaxFileSize),
			)
			continue
		case err != nil:
			return warnings, fmt.Errorf("read %s: %w", cf.File, err)
		}

		reps := make([]configfile.Replacement, 0, len(cf.Replace))
		for _, r := range cf.Replace {
			reps = append(reps, configfile.Replacement{
				Match:   r.Match,
				IfValue: r.IfValue,
				Value:   ResolvePlaceholders(r.ReplaceWith, gs, env),
				Type:    configfile.ValueType(firstNonEmpty(r.ValueType, "string")),
			})
		}
		out, write, err := configfile.Apply(cf.Parser, content.Bytes(), exists, reps, cfg)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to update %s (%s parser): %v", cf.File, cf.Parser, err))
			continue
		}
		if !write {
			continue
		}
		err = k.Exec(ctx, gs.Namespace, pod, FilesContainerName,
			[]string{"sh", "-c", PathScript + WriteFileScript, "sh", abs}, bytes.NewReader(out), nil)
		if err != nil {
			return warnings, fmt.Errorf("write %s: %w", cf.File, err)
		}
	}
	return warnings, nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
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
