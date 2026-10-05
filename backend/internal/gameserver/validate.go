package gameserver

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"app/api/v1alpha1"
	"app/internal/egg"
	"app/internal/validation"
)

// ValidateVariables checks the variables of a server against the rules of its egg; missing
// values use the egg default. The field errors are keyed by the variable name.
func ValidateVariables(e *v1alpha1.Egg, env map[string]string) error {
	errs := validation.Errors{}
	for _, v := range e.Spec.Variables {
		val, ok := env[v.EnvVariable]
		if !ok {
			val = v.DefaultValue
		}
		if err := egg.Validate(val, v.Rules); err != nil {
			errs[v.EnvVariable] = v.Name + " " + err.Error()
		}
	}
	return errs.OrNil()
}

// ValidatePorts requires at least one port, each in range and listed once.
func ValidatePorts(ports []int32) error {
	if len(ports) == 0 {
		return validation.Field("ports", errors.New("at least one port is required"))
	}
	seen := map[int32]bool{}
	for _, p := range ports {
		if p < 1 || p > 65535 {
			return validation.Field("ports", fmt.Errorf("port %d is out of range", p))
		}
		if seen[p] {
			return validation.Field("ports", fmt.Errorf("port %d is listed twice", p))
		}
		seen[p] = true
	}
	return nil
}

// ValidateResources checks the lower limits of memory, CPU and disk.
func ValidateResources(r v1alpha1.Resources) error {
	switch {
	case r.MemoryMiB < 64:
		return validation.Field("memoryMiB", errors.New("memory must be at least 64 MiB"))
	case r.CPUMillis < 0:
		return validation.Field("cpuMillis", errors.New("cpu must not be negative"))
	case r.DiskMiB < 256:
		return validation.Field("diskMiB", errors.New("disk must be at least 256 MiB"))
	}
	return nil
}

// NewName derives a DNS-1035 name ("survival-x7k2p") from the display name.
func NewName(display string) string {
	base := egg.Slug(display)
	if len(base) > 40 {
		base = strings.TrimRight(base[:40], "-")
	}
	if base == "" || base[0] < 'a' || base[0] > 'z' {
		base = "gs-" + base
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 5)
	for i := range suffix {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		suffix[i] = alphabet[n.Int64()]
	}
	return strings.TrimRight(base, "-") + "-" + string(suffix)
}
