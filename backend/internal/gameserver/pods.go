package gameserver

import (
	"cmp"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"app/api/v1alpha1"
)

// FilesPod is a small helper that mounts the data volume. It serves the file manager
// and runs the pre-start steps (config files). It only exists while it is used; the
// controller removes it after files.IdleTimeout.
//
// Every file operation runs as the game server user without capabilities on a read-only
// root file system: a symbolic link that users create in their files (e.g. to "/") gives
// them nothing the game server could not do already. Only the init container runs as
// root, to hand every file back to the game server user before the helper starts (egg images
// run as that user); it never follows symbolic links.
func FilesPod(gs *v1alpha1.GameServer, opts Options) *corev1.Pod {
	mounts := []corev1.VolumeMount{{Name: "data", MountPath: ServerRoot}}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FilesPodName(gs.Name),
			Namespace: gs.Namespace,
			Labels:    volumeLabels(gs.Name, RoleFiles),
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyAlways,
			Affinity:                      sameNodeAsVolume(gs.Name),
			TerminationGracePeriodSeconds: ptr.To[int64](1),
			EnableServiceLinks:            ptr.To(false),
			AutomountServiceAccountToken:  ptr.To(false),
			InitContainers: []corev1.Container{{
				Name:         "ownership",
				Image:        opts.HelperImage,
				Command:      []string{"chown", "-R", "-h", fmt.Sprintf("%d:%d", ServerUID, ServerGID), ServerRoot},
				VolumeMounts: mounts,
				Resources:    resources(500, 128),
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:                ptr.To[int64](0),
					AllowPrivilegeEscalation: ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(true),
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
						// CHOWN changes the owner, DAC_OVERRIDE reaches directories of other users.
						Add: []corev1.Capability{"CHOWN", "DAC_OVERRIDE"},
					},
				},
			}},
			Containers: []corev1.Container{{
				Name:         FilesContainerName,
				Image:        opts.HelperImage,
				Command:      []string{"sh", "-c", "trap 'exit 0' TERM INT; while :; do sleep 3600 & wait $!; done"},
				VolumeMounts: mounts,
				// File operations (tar, unzip) are short; 128 MiB is plenty for streaming.
				Resources:       resources(500, 128),
				SecurityContext: gameUserContext(),
			}},
			Volumes: []corev1.Volume{dataVolume(gs.Name)},
		},
	}
}

// gameUserContext runs a container as the game server user without any privileges.
func gameUserContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser:                ptr.To(ServerUID),
		RunAsGroup:               ptr.To(ServerGID),
		RunAsNonRoot:             ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}
}

// installShells can run the install script through a wrapper (sh -c).
var installShells = map[string]bool{"sh": true, "ash": true, "bash": true, "dash": true}

// installArgs runs the egg's install script. With a shell entrypoint (sh, ash, bash, dash) it
// afterwards hands the files the script created as root to the game server user (never
// following symbolic links) and keeps the script's exit code; other entrypoints run the script
// directly (the files pod's init container fixes the ownership then).
func installArgs(entrypoint string) []string {
	script := "/mnt/install/install.sh"
	if !installShells[entrypoint] {
		return []string{entrypoint, script}
	}
	return []string{
		entrypoint,
		"-c",
		fmt.Sprintf(`"$0" %s; rc=$?; chown -R -h %d:%d /mnt/server; exit $rc`, script, ServerUID, ServerGID),
		entrypoint,
	}
}

// InstallConfigMap holds the egg install script.
func InstallConfigMap(gs *v1alpha1.GameServer, e *v1alpha1.Egg) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      InstallConfigName(gs.Name),
			Namespace: gs.Namespace,
			Labels:    labels(gs.Name, RoleInstall),
		},
		Data: map[string]string{"install.sh": e.Spec.Install.Script},
	}
}

// InstallPod runs the egg install script as root with the data volume at /mnt/server.
func InstallPod(gs *v1alpha1.GameServer, e *v1alpha1.Egg, opts Options) *corev1.Pod {
	entrypoint := cmp.Or(e.Spec.Install.Entrypoint, "bash")
	memory := max(gs.Spec.Resources.MemoryMiB, 1024)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: InstallPodName(gs.Name), Namespace: gs.Namespace, Labels: volumeLabels(gs.Name, RoleInstall),
			Annotations: map[string]string{AnnotationInstallRevision: strconv.FormatInt(gs.Spec.InstallRevision, 10)},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                corev1.RestartPolicyNever,
			Affinity:                     sameNodeAsVolume(gs.Name),
			EnableServiceLinks:           ptr.To(false),
			AutomountServiceAccountToken: ptr.To(false),
			Hostname:                     "installer",
			Containers: []corev1.Container{{
				Name:  ContainerName,
				Image: cmp.Or(e.Spec.Install.Container, "ghcr.io/pelican-eggs/installers:debian"),
				// Only Cmd (not Entrypoint): the image's own entrypoint is kept.
				Args:  installArgs(entrypoint),
				Env:   EnvVars(Environment(gs, e, opts)),
				Stdin: true,
				TTY:   true,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "data", MountPath: "/mnt/server"},
					{Name: "script", MountPath: "/mnt/install"},
					{Name: "tmp", MountPath: "/tmp"},
				},
				Resources: resources(2000, memory),
				// Install scripts run as root (package managers, chown), but only with the
				// capabilities file management and package installs need.
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
						Add: []corev1.Capability{
							"CHOWN",
							"DAC_OVERRIDE",
							"FOWNER",
							"FSETID",
							"SETUID",
							"SETGID",
							"KILL",
						},
					},
				},
			}},
			Volumes: []corev1.Volume{
				dataVolume(gs.Name),
				{Name: "script", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: InstallConfigName(gs.Name)},
					DefaultMode:          ptr.To[int32](0o755),
				}}},
				tmpVolume(),
			},
		},
	}
	return pod
}

// GamePod runs the server image: fixed user, read-only root file system, TTY and stdin.
func GamePod(gs *v1alpha1.GameServer, e *v1alpha1.Egg, opts Options) *corev1.Pod {
	res := resources(gs.Spec.Resources.CPUMillis, memoryLimitMiB(gs.Spec.Resources.MemoryMiB))
	if gs.Spec.Resources.CPUMillis == 0 {
		// Unlimited CPU: no limit, but a fixed reservation for the scheduler.
		delete(res.Limits, corev1.ResourceCPU)
		res.Requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(UnlimitedCPURequestMillis, resource.DecimalSI)
	}
	var ports []corev1.ContainerPort
	for _, p := range gs.Spec.Ports {
		ports = append(ports,
			corev1.ContainerPort{Name: fmt.Sprintf("tcp-%d", p), ContainerPort: p, Protocol: corev1.ProtocolTCP},
			corev1.ContainerPort{Name: fmt.Sprintf("udp-%d", p), ContainerPort: p, Protocol: corev1.ProtocolUDP},
		)
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: GamePodName(gs.Name), Namespace: gs.Namespace, Labels: volumeLabels(gs.Name, RoleGame),
			Annotations: map[string]string{AnnotationRuntimeHash: RuntimeHash(gs, e, opts)},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyNever,
			Affinity:                      sameNodeAsVolume(gs.Name),
			EnableServiceLinks:            ptr.To(false),
			AutomountServiceAccountToken:  ptr.To(false),
			TerminationGracePeriodSeconds: ptr.To(max(gs.Spec.StopTimeoutSeconds, 30)),
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:      ptr.To(ServerUID),
				RunAsGroup:     ptr.To(ServerGID),
				RunAsNonRoot:   ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
			Containers: []corev1.Container{{
				Name:       ContainerName,
				Image:      gs.Spec.Image,
				Env:        EnvVars(Environment(gs, e, opts)),
				WorkingDir: ServerRoot,
				Stdin:      true,
				TTY:        true,
				Ports:      ports,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "data", MountPath: ServerRoot},
					{Name: "tmp", MountPath: "/tmp"},
				},
				Resources: res,
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(true),
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				},
			}},
			Volumes: []corev1.Volume{dataVolume(gs.Name), tmpVolume()},
		},
	}
}

// PodReady reports whether the pod runs and all its containers are ready.
func PodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
