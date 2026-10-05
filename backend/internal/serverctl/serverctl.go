// Package serverctl performs power actions and console commands on game servers. It is
// shared by the REST API, the console websocket and the schedule runner.
package serverctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/console"
	"app/internal/files"
	"app/internal/gameserver"
	"app/internal/kube"
)

// Errors of power actions and commands.
var (
	ErrOffline   = errors.New("server is not running")
	ErrSuspended = errors.New("this server is suspended by an administrator")
	ErrRunning   = errors.New("stop the server first")
)

// Signals are the power actions.
var Signals = []string{"start", "stop", "restart", "kill"}

// Ops runs power actions and commands.
type Ops struct {
	Client client.Client
	// Reader reads uncached (transfers wait for objects to be gone).
	Reader client.Reader
	Kube   *kube.Client
	Hub    *console.Hub
	// Namespace is the panel namespace, which holds the eggs.
	Namespace string
	// Files reports running restores (a server must not start while its files are replaced).
	Files *files.Service
}

// Power applies start, stop, restart or kill. The controller picks the change up; call
// its trigger afterwards for an immediate reconcile.
func (o *Ops) Power(ctx context.Context, gs *v1alpha1.GameServer, signal string) error {
	if signal == "start" || signal == "restart" {
		if gs.Spec.Suspended {
			return ErrSuspended
		}
		if o.Files != nil && o.Files.Busy(files.RefOf(gs), files.JobRestore) {
			return files.ErrRestoring
		}
	}
	patch := client.MergeFrom(gs.DeepCopy())
	switch signal {
	case "start":
		gs.Spec.State = v1alpha1.PowerRunning
	case "stop":
		gs.Spec.State = v1alpha1.PowerStopped
	case "restart":
		if gs.Status.PodUID != "" {
			if gs.Annotations == nil {
				gs.Annotations = map[string]string{}
			}
			gs.Annotations[gameserver.AnnotationRestart] = time.Now().UTC().Format(time.RFC3339)
		}
		gs.Spec.State = v1alpha1.PowerRunning
	case "kill":
		gs.Spec.State = v1alpha1.PowerStopped
		delete(gs.Annotations, gameserver.AnnotationRestart)
	default:
		return fmt.Errorf("unknown power signal %q", signal)
	}
	if err := o.Client.Patch(ctx, gs, patch); err != nil {
		return err
	}
	if signal == "kill" {
		o.Hub.Daemon(gs.Name, "Server marked as stopping...")
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: gameserver.GamePodName(gs.Name), Namespace: gs.Namespace},
		}
		if err := o.Client.Delete(ctx, pod, client.GracePeriodSeconds(0)); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

// Command writes a command to the server console. Sending the egg stop command is
// treated as a stop request.
func (o *Ops) Command(ctx context.Context, gs *v1alpha1.GameServer, command string) error {
	if gs.Spec.Suspended {
		return ErrSuspended
	}
	if gs.Status.PodUID == "" ||
		(gs.Status.Phase != v1alpha1.PhaseRunning && gs.Status.Phase != v1alpha1.PhaseStarting) {
		return ErrOffline
	}
	command = strings.TrimRight(command, "\r\n")
	// Eggs live in the panel namespace, not in the namespace of the server.
	e := &v1alpha1.Egg{}
	if err := o.Client.Get(ctx, client.ObjectKey{Namespace: o.Namespace, Name: gs.Spec.EggRef}, e); err == nil &&
		e.Spec.Stop != "" && !strings.HasPrefix(e.Spec.Stop, "^") && command == e.Spec.Stop {
		patch := client.MergeFrom(gs.DeepCopy())
		gs.Spec.State = v1alpha1.PowerStopped
		if gs.Annotations == nil {
			gs.Annotations = map[string]string{}
		}
		gs.Annotations[gameserver.AnnotationStopSent] = gs.Status.PodUID
		if err := o.Client.Patch(ctx, gs, patch); err != nil {
			return err
		}
	}
	return o.Kube.AttachWrite(
		ctx, gs.Namespace, gameserver.GamePodName(gs.Name), gameserver.ContainerName, []byte(command+"\n"),
	)
}

// Reinstall stops the server and runs the egg's install script again (new install revision).
func (o *Ops) Reinstall(ctx context.Context, gs *v1alpha1.GameServer) error {
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.InstallRevision = max(gs.Spec.InstallRevision, gs.Status.InstalledRevision) + 1
	gs.Spec.State = v1alpha1.PowerStopped
	return o.Client.Patch(ctx, gs, patch)
}

// Suspend stops and locks a server for its owner (suspended=true) or unlocks it.
func (o *Ops) Suspend(ctx context.Context, gs *v1alpha1.GameServer, suspended bool) error {
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.Suspended = suspended
	msg := "Server unsuspended."
	if suspended {
		gs.Spec.State = v1alpha1.PowerStopped
		msg = "Server suspended by an administrator."
	}
	if err := o.Client.Patch(ctx, gs, patch); err != nil {
		return err
	}
	o.Hub.Daemon(gs.Name, msg)
	return nil
}

// Stopped fails with ErrRunning unless the server is stopped and not installing, e.g. before
// its files are replaced by a backup.
func Stopped(gs *v1alpha1.GameServer) error {
	if gs.Status.PodUID != "" || gs.Spec.State == v1alpha1.PowerRunning || gs.Status.Phase == v1alpha1.PhaseInstalling {
		return ErrRunning
	}
	return nil
}
