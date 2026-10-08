package gameserver

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"app/api/v1alpha1"
	"app/internal/egg"
)

// StartupCommand returns the effective (unsubstituted) startup command: the server's own, else the egg
// command it picked, else the egg's default.
func StartupCommand(gs *v1alpha1.GameServer, e *v1alpha1.Egg) string {
	if gs.Spec.Startup != "" {
		return gs.Spec.Startup
	}
	for _, c := range e.Spec.StartupCommands {
		if c.Name == gs.Spec.StartupName {
			return c.Command
		}
	}
	return e.Spec.Startup
}

// PrimaryPort is the first allocation (SERVER_PORT).
func PrimaryPort(gs *v1alpha1.GameServer) int32 {
	if len(gs.Spec.Ports) == 0 {
		return 0
	}
	return gs.Spec.Ports[0]
}

// Environment returns the environment of the server as eggs expect it:
// egg variables (value or default) plus the system variables, which always win.
func Environment(gs *v1alpha1.GameServer, e *v1alpha1.Egg, opts Options) map[string]string {
	env := map[string]string{}
	for _, v := range e.Spec.Variables {
		if egg.ReservedVariables[v.EnvVariable] {
			continue
		}
		val := v.DefaultValue
		if cur, ok := gs.Spec.Environment[v.EnvVariable]; ok {
			val = cur
		}
		env[v.EnvVariable] = val
	}
	tz := opts.Timezone
	if tz == "" {
		tz = "UTC"
	}
	env["TZ"] = tz
	env["STARTUP"] = StartupCommand(gs, e)
	env["SERVER_MEMORY"] = strconv.FormatInt(gs.Spec.Resources.MemoryMiB, 10)
	// Pods bind to all interfaces; the load balancer forwards to the pod IP.
	env["SERVER_IP"] = "0.0.0.0"
	env["SERVER_PORT"] = strconv.Itoa(int(PrimaryPort(gs)))
	env["P_SERVER_UUID"] = string(gs.UID)
	env["P_SERVER_LOCATION"] = "kubernetes"
	env["P_SERVER_ALLOCATION_LIMIT"] = "0"
	return env
}

// EnvVars converts the environment into a sorted list for pod specs.
func EnvVars(env map[string]string) []corev1.EnvVar {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]corev1.EnvVar, 0, len(keys))
	for _, k := range keys {
		out = append(out, corev1.EnvVar{Name: k, Value: env[k]})
	}
	return out
}

var placeholderRe = regexp.MustCompile(`{{\s*([\w.-]+)\s*}}`)

// ResolvePlaceholders substitutes the panel side placeholders used in egg config
// files, in both the Pterodactyl and the Pelican dialect. {{config.*}} is left for
// the config file parser. Unknown placeholders are kept unchanged.
func ResolvePlaceholders(value string, gs *v1alpha1.GameServer, env map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(value, func(m string) string {
		key := placeholderRe.FindStringSubmatch(m)[1]
		switch key {
		case "server.build.default.port", "server.allocations.default.port":
			return strconv.Itoa(int(PrimaryPort(gs)))
		case "server.build.default.ip", "server.allocations.default.ip":
			return "0.0.0.0"
		case "server.build.memory", "server.build.memory_limit":
			return strconv.FormatInt(gs.Spec.Resources.MemoryMiB, 10)
		case "server.build.disk", "server.build.disk_space":
			return strconv.FormatInt(gs.Spec.Resources.DiskMiB, 10)
		case "server.uuid":
			return string(gs.UID)
		}
		for _, prefix := range []string{"server.build.env.", "server.environment.", "env."} {
			if name, ok := strings.CutPrefix(key, prefix); ok {
				// Legacy {{env.SERVER_PORT}} etc. map to the system values.
				if v, ok := env[name]; ok {
					return v
				}
				return m
			}
		}
		return m
	})
}
