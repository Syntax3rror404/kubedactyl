// Package gameserver translates GameServer and Egg resources into Kubernetes objects.
package gameserver

// Options are cluster specific settings of the panel.
type Options struct {
	// Namespace is the panel namespace (eggs, users, settings); servers live in the user namespaces.
	Namespace string
	// StorageClass is enabled in the panel settings on the first start and used for
	// servers without a storage class.
	StorageClass string
	// LoadBalancerPool is enabled in the panel settings on the first start (pool name or
	// the value of its lb.cilium.io/pool selector label).
	LoadBalancerPool string
	// HelperImage runs the file manager pod and the pre-start steps.
	HelperImage string
	// DockerInterface is the value of {{config.docker.interface}} in egg config files.
	DockerInterface string
	// Timezone is passed to servers as TZ.
	Timezone string
}

// Labels and annotations used on child objects.
const (
	LabelServer = "kubedactyl.io/server"
	LabelRole   = "kubedactyl.io/role"
	// LabelVolume marks pods that mount the data volume of the server (see sameNodeAsVolume).
	LabelVolume = "kubedactyl.io/volume"

	RoleGame    = "game"
	RoleInstall = "install"
	RoleFiles   = "files"

	AnnotationInstallRevision = "kubedactyl.io/install-revision"
	// AnnotationRestart requests a restart of a running server.
	AnnotationRestart = "kubedactyl.io/restart"
	// AnnotationStopSent marks that the stop command was typed into the console of the
	// pod with this UID, so the controller must not send it again.
	AnnotationStopSent = "kubedactyl.io/stop-sent"
	// AnnotationExitHandled marks a terminated game pod whose exit was processed. The pod
	// is kept until the next start so its console output stays available.
	AnnotationExitHandled = "kubedactyl.io/exit-handled"
	// AnnotationFixedIPs asks Cilium LB IPAM for the server's fixed IPs (comma separated).
	AnnotationFixedIPs = "lbipam.cilium.io/ips"
	// AnnotationPoolLabels lists the service label keys set for the load balancer pool,
	// so they can be removed when the server moves to another pool.
	AnnotationPoolLabels = "kubedactyl.io/pool-labels"
	// AnnotationRuntimeHash records the runtime settings a game pod was created with (v2: also
	// the egg's startup, variable defaults and config files; pods with only the v1 hash from older
	// versions never ask for a restart).
	AnnotationRuntimeHash = "kubedactyl.io/runtime-hash-v2"
	// AnnotationKeepVolume keeps the data volume when the server object is deleted, because a
	// transfer to another owner moves it (serverctl.Transfer).
	AnnotationKeepVolume = "kubedactyl.io/keep-volume"
	// AnnotationMovedVolume names the persistent volume a transferred server's claim binds to.
	AnnotationMovedVolume = "kubedactyl.io/moved-volume"
	// AnnotationInstalledRevision carries the install state of a transferred server, whose
	// status starts empty: it must not be installed again.
	AnnotationInstalledRevision = "kubedactyl.io/installed-revision"

	// PathScript defines shell functions for scripts that touch the server files. real prints a
	// path with every symbolic link resolved (also links to something missing, which a write
	// would create; BusyBox realpath resolves their relative targets against the working folder,
	// so real follows them itself) and missing parts appended as given (nothing for link loops
	// or ".." after a missing part); entry resolves only its folder (deleting or moving a link
	// touches the link); inside ends the script as not found (exit 3) for a path outside the
	// server root, so links cannot lead out of the volume. BusyBox realpath takes no "--"; the
	// paths are always absolute.
	PathScript = `real() {
  local p="$1" s="" r n=0
  while [ "$n" -lt 64 ]; do
    n=$((n + 1))
    if [ -L "$p" ] && [ ! -e "$p" ]; then
      r=$(readlink "$p"); case "$r" in /*) p="$r" ;; *) p="$(dirname -- "$p")/$r" ;; esac
    elif r=$(realpath "$p" 2>/dev/null); then
      case "$s" in */..|*/../*) return ;; esac
      echo "$r$s"
      return
    else
      s="/${p##*/}$s"; p=$(dirname -- "$p")
    fi
  done
}
entry() { echo "$(real "$(dirname -- "$1")")/${1##*/}"; }
inside() { case "$1" in ` + ServerRoot + `|` + ServerRoot + `/*) ;; *) exit 3 ;; esac; }
`

	// WriteFileScript writes stdin to the file "$1" (creating its folder) inside a pod; it needs
	// PathScript before it. cat must not be the last command: busybox sh then replaces itself
	// with cat, and the data sent to stdin through the Kubernetes API does not reach the file
	// (empty or cut short).
	WriteFileScript = `inside "$(real "$1")"
mkdir -p "$(dirname -- "$1")" && cat > "$1" || exit 1`

	// ContainerName is the name of the main container in game and install pods.
	ContainerName = "server"
	// FilesContainerName is the name of the container in the files pod.
	FilesContainerName = "files"
	// ServerRoot is where the server data volume is mounted in game and helper pods.
	ServerRoot = "/home/container"

	// UID/GID of the game process (egg images expect this user).
	ServerUID int64 = 988
	ServerGID int64 = 988

	Finalizer = "kubedactyl.io/cleanup"
)

// Object names derived from the game server name.
func PVCName(server string) string           { return server + "-data" }
func ServiceName(server string) string       { return server }
func GamePodName(server string) string       { return server + "-game" }
func InstallPodName(server string) string    { return server + "-install" }
func FilesPodName(server string) string      { return server + "-files" }
func InstallConfigName(server string) string { return server + "-install" }
