package serverctl

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/files"
	"app/internal/gameserver"
)

// Errors of storage migrations.
var (
	ErrMigrating        = errors.New("the server files are moving to another storage class")
	ErrSameStorageClass = errors.New("the server already uses this storage class")
	ErrInstalling       = errors.New("wait until the installation has finished")
	ErrNotMigrating     = errors.New("no storage migration is running")
	ErrSwitching        = errors.New("the files are copied already, the migration finishes in a moment")
)

// Migrate moves the server files to a volume of another storage class: it names the class in the
// spec; the controller stops the server (it is locked while migrating), copies the files and
// switches the volume. With start the server stays wanted running, so it starts when the
// migration has ended.
func (o *Ops) Migrate(ctx context.Context, gs *v1alpha1.GameServer, storageClass string, start bool) error {
	switch {
	case gameserver.Migrating(gs):
		return ErrMigrating
	case gs.Status.StorageClass == "":
		return ErrNoVolume
	case gs.Status.StorageClass == storageClass:
		return ErrSameStorageClass
	case gs.Status.Phase == v1alpha1.PhaseInstalling:
		return ErrInstalling
	case o.Files != nil && o.Files.Busy(
		files.RefOf(gs), files.JobPull, files.JobBackup, files.JobRestore, files.JobCompress, files.JobDecompress,
	):
		return ErrFilesBusy
	}
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.StorageClass = storageClass
	gs.Spec.State = v1alpha1.PowerStopped
	if start {
		gs.Spec.State = v1alpha1.PowerRunning
	}
	return o.Client.Patch(ctx, gs, patch)
}

// CancelMigration names the storage class of the current volume again, which ends a migration
// before the switch; the copy is deleted.
func (o *Ops) CancelMigration(ctx context.Context, gs *v1alpha1.GameServer) error {
	if !gameserver.Migrating(gs) {
		return ErrNotMigrating
	}
	if m := gs.Status.Migration; m != nil && m.Step == v1alpha1.MigrationSwitching {
		return ErrSwitching
	}
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.StorageClass = gs.Status.StorageClass
	return o.Client.Patch(ctx, gs, patch)
}
