package gameserver

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"app/api/v1alpha1"
)

// Migrating tells whether the server files are moving to a volume of another storage class: the
// spec names another class than the bound volume has.
func Migrating(gs *v1alpha1.GameServer) bool {
	return gs.Status.StorageClass != "" && gs.Spec.StorageClass != "" && gs.Spec.StorageClass != gs.Status.StorageClass
}

// migrateScript copies everything from /from to /to with owners, modes and links, then checks that
// both hold the same number of entries and bytes. It prints "progress DONE TOTAL" (files) every
// 100 files and "verified FILES BYTES" at the end; tar errors go to stderr. The archive streams from
// one tar into the other, which writes the names unbuffered only while the archive goes to stdout.
const migrateScript = `set -o pipefail
count() { find . ! -type d | wc -l; }
size() { find . -type f -exec stat -c %s {} + | awk '{ s += $1 } END { print s + 0 }'; }
cd /from || exit 1
total=$(count)
echo "progress 0 $total"
{ tar -cvf - . 2>&3 | tar -xpf - -C /to 2>&3; } 3>&1 | awk -v total="$total" '
  /^\.\// { if ($0 !~ /\/$/ && ++n % 100 == 0) { print "progress", n, total; fflush() } next }
  { print > "/dev/stderr" }' || exit 1
echo "progress $total $total"
from="$(count) $(size)"
cd /to || exit 1
[ "$from" = "$(count) $(size)" ] || { echo "the copy differs from the original files" >&2; exit 1; }
echo "verified $from"`

// MigratePod copies the server files from the data volume to the new claim of a storage migration.
// It runs as root to read every file and keep its owner, with only the capabilities that needs.
func MigratePod(gs *v1alpha1.GameServer, opts Options) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: MigrateName(gs.Name), Namespace: gs.Namespace, Labels: volumeLabels(gs.Name, RoleMigrate),
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                corev1.RestartPolicyNever,
			Affinity:                     sameNodeAsVolume(gs.Name),
			EnableServiceLinks:           ptr.To(false),
			AutomountServiceAccountToken: ptr.To(false),
			Containers: []corev1.Container{{
				Name:    ContainerName,
				Image:   opts.HelperImage,
				Command: []string{"sh", "-c", migrateScript},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "data", MountPath: "/from", ReadOnly: true},
					{Name: "target", MountPath: "/to"},
				},
				Resources: resources(1000, 256),
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:                ptr.To[int64](0),
					AllowPrivilegeEscalation: ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(true),
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
						// Read every file, keep owners, modes and times (also setgid bits).
						Add: []corev1.Capability{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID"},
					},
				},
			}},
			Volumes: []corev1.Volume{
				dataVolume(gs.Name),
				{Name: "target", VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: MigrateName(gs.Name)},
				}},
			},
		},
	}
}

// MigrateClaim is the new data volume of a storage migration, of the class the server moves to.
func MigrateClaim(gs *v1alpha1.GameServer, opts Options) *corev1.PersistentVolumeClaim {
	pvc := PVC(gs, opts)
	pvc.Name = MigrateName(gs.Name)
	pvc.Labels = labels(gs.Name, RoleMigrate)
	pvc.Spec.VolumeName = ""
	return pvc
}

// MigrateLog reads the output of the migrate pod: the last progress and, when it failed, the last
// line that is no progress report (tar's or the check's error).
func MigrateLog(log string) (done, total int64, problem string) {
	for line := range strings.Lines(log) {
		f := strings.Fields(line)
		switch {
		case len(f) == 3 && f[0] == "progress":
			done, _ = strconv.ParseInt(f[1], 10, 64)
			total, _ = strconv.ParseInt(f[2], 10, 64)
		case len(f) > 0 && f[0] != "verified":
			problem = strings.TrimSpace(line)
		}
	}
	return done, total, problem
}
