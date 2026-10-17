package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PowerState is the desired power state of a game server.
// +kubebuilder:validation:Enum=Running;Stopped
type PowerState string

const (
	PowerRunning PowerState = "Running"
	PowerStopped PowerState = "Stopped"
)

// TrafficPolicy is the external traffic policy of a game server's load balancer service.
// +kubebuilder:validation:Enum=Local;Cluster
type TrafficPolicy string

const (
	// TrafficLocal keeps the players' source IPs visible to the game server (ban lists, logs);
	// only the node running the server answers.
	TrafficLocal TrafficPolicy = "Local"
	// TrafficCluster lets every node answer and forward the traffic; the game server sees a
	// node IP instead of the player's.
	TrafficCluster TrafficPolicy = "Cluster"
)

// ShareAccess is what a subuser may do on a server shared with them.
// +kubebuilder:validation:Enum=view;control
type ShareAccess string

const (
	// ShareView shows status, address and console output.
	ShareView ShareAccess = "view"
	// ShareControl also allows start, stop and console commands.
	ShareControl ShareAccess = "control"
)

// ServerShare gives one subuser of the owner access to the server.
type ServerShare struct {
	// User is the username of a subuser of the server's owner.
	User string `json:"user"`
	// +kubebuilder:default=view
	Access ShareAccess `json:"access"`
}

// ServiceType is how a game server is published.
// +kubebuilder:validation:Enum=LoadBalancer;ClusterIP
type ServiceType string

const (
	// ServiceLoadBalancer gives the server an address of its load balancer pool.
	ServiceLoadBalancer ServiceType = "LoadBalancer"
	// ServiceClusterIP publishes the server only inside the cluster (e.g. behind a proxy server);
	// it is reached through its service's DNS name.
	ServiceClusterIP ServiceType = "ClusterIP"
)

// Phase is the observed state of a game server.
type Phase string

const (
	PhasePending       Phase = "Pending"
	PhaseInstalling    Phase = "Installing"
	PhaseInstallFailed Phase = "InstallFailed"
	PhaseOffline       Phase = "Offline"
	PhaseStarting      Phase = "Starting"
	PhaseRunning       Phase = "Running"
	PhaseStopping      Phase = "Stopping"
)

// Resources of a game server.
type Resources struct {
	// MemoryMiB is the memory of the server (MiB), also passed to it as SERVER_MEMORY.
	// +kubebuilder:validation:Minimum=64
	MemoryMiB int64 `json:"memoryMiB"`
	// CPUMillis limits the CPU of the server (millicores, 1000 = one core). 0 = unlimited.
	// +optional
	CPUMillis int64 `json:"cpuMillis,omitempty"`
	// DiskMiB is the size of the server's volume (MiB).
	// +kubebuilder:validation:Minimum=256
	DiskMiB int64 `json:"diskMiB"`
}

// GameServerSpec defines the desired state of a game server.
type GameServerSpec struct {
	// DisplayName is the name shown in the panel. Empty: uses the object name.
	DisplayName string `json:"displayName,omitempty"`
	// EggRef is the name of the Egg in the same namespace.
	EggRef string `json:"eggRef"`
	// Image is the runtime image (one of the egg's docker images or a custom one).
	Image string `json:"image"`
	// Startup replaces the egg's startup command. Empty: uses StartupName.
	// +optional
	Startup string `json:"startup,omitempty"`
	// StartupName picks one of the egg's startup commands by name. Empty or unknown: uses the egg's
	// default. Ignored when Startup is set.
	// +optional
	StartupName string `json:"startupName,omitempty"`
	// Environment holds the values of the egg variables. Missing ones use the egg's defaults.
	// +optional
	Environment map[string]string `json:"environment,omitempty"`
	Resources   Resources         `json:"resources"`
	// Ports are published as TCP and UDP. The first one is the primary port (SERVER_PORT).
	// +kubebuilder:validation:MinItems=1
	Ports []int32 `json:"ports"`
	// StorageClass of the server's volume. Changing it moves the files to a new volume of that class
	// (status.migration); the server is locked meanwhile. Admins only.
	// +optional
	StorageClass string `json:"storageClass,omitempty"`
	// ServiceType publishes the server through a load balancer or only inside the cluster (ClusterIP).
	// Empty: LoadBalancer. A ClusterIP server keeps its pool for when it gets a load balancer again.
	// +optional
	ServiceType ServiceType `json:"serviceType,omitempty"`
	// LoadBalancerPool is the Cilium LB IPAM pool the address comes from. The service gets the labels
	// of the pool's service selector.
	// +optional
	LoadBalancerPool string `json:"loadBalancerPool,omitempty"`
	// LoadBalancerIP fixes the server's addresses: one IP, or one per IP family separated by a comma,
	// from the load balancer pool. Empty: the pool picks.
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
	// ExternalTrafficPolicy of the server's service. Empty: uses Local. Admins only.
	// +optional
	ExternalTrafficPolicy TrafficPolicy `json:"externalTrafficPolicy,omitempty"`
	// IPv6 gives the server an address of each IP family the cluster has (PreferDualStack). Only with
	// a dual stack cluster and a pool with an IPv6 block. Admins only.
	// +optional
	IPv6 bool `json:"ipv6,omitempty"`
	// State is the power state the server should be in.
	// +kubebuilder:default=Stopped
	State PowerState `json:"state,omitempty"`
	// InstallRevision reinstalls the server whenever it differs from status.installedRevision.
	// +kubebuilder:default=1
	InstallRevision int64 `json:"installRevision,omitempty"`
	// SkipInstall creates the server without running the egg's install script.
	// +optional
	SkipInstall bool `json:"skipInstall,omitempty"`
	// CrashRestart restarts the server after a crash (not twice within 60 seconds).
	// +kubebuilder:default=true
	CrashRestart *bool `json:"crashRestart,omitempty"`
	// StopTimeoutSeconds is how long the server may take to stop after the stop command before it
	// is killed (seconds).
	// +kubebuilder:default=600
	StopTimeoutSeconds int64 `json:"stopTimeoutSeconds,omitempty"`
	// Suspended stops the server and locks it for its owner. Admins only.
	// +optional
	Suspended bool `json:"suspended,omitempty"`
	// SharedWith are the subusers of the owner who may see the server. Files, backups and settings stay
	// with the owner.
	// +optional
	// +listType=map
	// +listMapKey=user
	// +kubebuilder:validation:MaxItems=100
	SharedWith []ServerShare `json:"sharedWith,omitempty"`
	// Schedules run tasks at fixed times or events of the server.
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=20
	Schedules []Schedule `json:"schedules,omitempty"`
}

// Schedule runs its tasks one after another whenever the cron expression matches or, with
// Event set, at a moment of the server's life (shown as "Tasks" in the web interface).
type Schedule struct {
	// Name identifies the schedule within the server.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=50
	Name string `json:"name"`
	// Cron is a five-field cron expression ("0 4 * * *") or a descriptor like "@daily",
	// evaluated in the panel's time zone. Either Cron or Event is set.
	// +optional
	Cron string `json:"cron,omitempty"`
	// Event runs the schedule when the server was marked as running ("started") or before it
	// is stopped or restarted ("stopping"; the stop waits until the tasks are done).
	// +optional
	// +kubebuilder:validation:Enum=started;stopping
	Event string `json:"event,omitempty" enums:"started,stopping"`
	// Enabled schedules run automatically; disabled ones can still be run by hand.
	Enabled bool `json:"enabled"`
	// OnlyWhenOnline skips the run when the server is not running.
	// +optional
	OnlyWhenOnline bool `json:"onlyWhenOnline,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	Tasks []ScheduleTask `json:"tasks"`
}

// ScheduleTask is one step of a schedule.
type ScheduleTask struct {
	// Action is a console command, a power action or a backup.
	// +kubebuilder:validation:Enum=command;start;stop;restart;kill;backup
	Action string `json:"action" enums:"command,start,stop,restart,kill,backup"`
	// Payload is the console command (action "command") or the backup label (action "backup").
	// +optional
	// +kubebuilder:validation:MaxLength=500
	Payload string `json:"payload,omitempty"`
	// DelaySeconds waits before this task, counted from the end of the previous one (seconds).
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=900
	DelaySeconds int32 `json:"delaySeconds,omitempty"`
}

// ScheduleStatus is the result of the last run of a schedule.
type ScheduleStatus struct {
	Name      string       `json:"name"`
	LastRunAt *metav1.Time `json:"lastRunAt,omitempty"`
	// LastResult is "ok", "skipped: <reason>" or "failed: <reason>".
	LastResult string `json:"lastResult,omitempty"`
}

// Steps of a storage migration.
const (
	MigrationPreparing = "Preparing"
	MigrationCopying   = "Copying"
	MigrationSwitching = "Switching"
	MigrationFailed    = "Failed"
)

// StorageMigration moves the files of a stopped server to a volume of another storage class: a
// pod copies them and checks the copy, then the server's claim is bound to the new volume. The old
// volume is deleted only once the server uses the new one.
type StorageMigration struct {
	// From is the storage class the server moves away from, To the one it moves to.
	From string `json:"from"`
	To   string `json:"to"`
	// Step is Preparing (stopping the server, creating the new volume), Copying, Switching (binding
	// the server to the new volume) or Failed.
	// +kubebuilder:validation:Enum=Preparing;Copying;Switching;Failed
	Step      string      `json:"step"      enums:"Preparing,Copying,Switching,Failed"`
	StartedAt metav1.Time `json:"startedAt"`
	// Done is the number of files copied so far, Total the number of all files.
	// +optional
	Done int64 `json:"done,omitempty"`
	// +optional
	Total int64 `json:"total,omitempty"`
	// Volume is the new persistent volume, PreviousVolume the old one (both set when switching).
	// +optional
	Volume string `json:"volume,omitempty"`
	// +optional
	PreviousVolume string `json:"previousVolume,omitempty"`
	// Error tells why the migration failed. The server stays on its old volume.
	// +optional
	Error string `json:"error,omitempty"`
}

// GameServerStatus defines the observed state of a game server.
type GameServerStatus struct {
	Phase   Phase  `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
	// Address is the external IP assigned by the load balancer (the first of Addresses), or the
	// service's DNS name inside the cluster (<service>.<namespace>.svc) for a ClusterIP server.
	Address string `json:"address,omitempty"`
	// Addresses are all external IPs assigned by the load balancer (one per IP family).
	// +optional
	Addresses []string `json:"addresses,omitempty"`
	// InstalledRevision is the last completed install revision.
	InstalledRevision int64 `json:"installedRevision,omitempty"`
	// InstallExitCode is the exit code of the last install script run.
	InstallExitCode *int32 `json:"installExitCode,omitempty"`
	// PodUID identifies the current game pod.
	PodUID string `json:"podUID,omitempty"`
	// StartedAt is when the current game pod was created.
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// StopRequestedAt is set when a graceful stop was sent to the current pod.
	StopRequestedAt *metav1.Time `json:"stopRequestedAt,omitempty"`
	// StopTasksStartedAt is set while the "stopping" schedules of the current pod run; the
	// stop is sent when they are done.
	// +optional
	StopTasksStartedAt *metav1.Time `json:"stopTasksStartedAt,omitempty"`
	// LastCrashAt is the time of the last detected crash.
	LastCrashAt *metav1.Time `json:"lastCrashAt,omitempty"`
	// LastExitCode is the exit code of the last terminated game process.
	LastExitCode *int32 `json:"lastExitCode,omitempty"`
	// Schedules holds the last run of every schedule.
	// +optional
	// +listType=map
	// +listMapKey=name
	Schedules []ScheduleStatus `json:"schedules,omitempty"`
	// RestartRequired is set while the running game pod was created with other runtime
	// settings (image, startup, variables, memory, CPU, ports) than the spec has now.
	RestartRequired bool `json:"restartRequired,omitempty"`
	// StorageClass of the bound volume. While spec.storageClass differs, the server is migrating.
	// +optional
	StorageClass string `json:"storageClass,omitempty"`
	// Migration is the running or last failed storage migration.
	// +optional
	Migration *StorageMigration `json:"migration,omitempty"`
	// DiskUsedBytes is the last measured size of the server files (bytes). It is measured only
	// while the files pod runs, which happens on demand.
	DiskUsedBytes      *int64       `json:"diskUsedBytes,omitempty"`
	DiskMeasuredAt     *metav1.Time `json:"diskMeasuredAt,omitempty"`
	ObservedGeneration int64        `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kgs
// +kubebuilder:printcolumn:name="Egg",type=string,JSONPath=`.spec.eggRef`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.spec.state`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Suspended",type=boolean,JSONPath=`.spec.suspended`
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.address`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GameServer is a game server instance created from an Egg.
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec"`
	Status GameServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameServerList contains a list of GameServers.
type GameServerList struct {
	metav1.TypeMeta `             json:",inline"`
	metav1.ListMeta `             json:"metadata,omitempty"`
	Items           []GameServer `json:"items"`
}

func init() {
	register(&GameServer{}, &GameServerList{})
}
