// Package checks is the result type of every check the panel shows: the cluster health in the
// sidebar and the diagnostics of a server. The web interface shows them with the same icons.
package checks

// Status is the result of one check.
type Status string

const (
	OK      Status = "ok"
	Warning Status = "warning"
	Error   Status = "error"
	// Skipped means the check could not run (e.g. the server is stopped); it counts as OK.
	Skipped Status = "skipped"
)

// Check is one check with its result.
type Check struct {
	ID      string `json:"id"                example:"metrics"`
	Label   string `json:"label"             example:"Metrics server"`
	Status  Status `json:"status"                                                                                enums:"ok,warning,error,skipped"`
	Message string `json:"message,omitempty" example:"metrics.k8s.io is not available, no CPU and memory usage"`
}

// New returns a check.
func New(id, label string, status Status, message string) Check {
	return Check{ID: id, Label: label, Status: status, Message: message}
}

// Worst is the most severe status of the checks (OK when there are none).
func Worst(list []Check) Status {
	worst := OK
	for _, c := range list {
		if c.Status == Error || (c.Status == Warning && worst == OK) {
			worst = c.Status
		}
	}
	return worst
}
