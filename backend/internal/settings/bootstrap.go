package settings

import (
	"cmp"
	"context"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/gameserver"
	"app/internal/tenancy"
)

// Bootstrap creates the settings on the first start: the initial storage class and pool
// come from the command line. The pool may be given by name or by the value of its
// lb.cilium.io/pool service selector label. Servers created before these settings
// existed get their storage class and pool filled in.
func Bootstrap(ctx context.Context, c client.Client, ns, storageClass, pool string, log *slog.Logger) error {
	obj := &v1alpha1.PanelSettings{}
	err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: v1alpha1.SettingsName}, obj)
	if apierrors.IsNotFound(err) {
		obj = &v1alpha1.PanelSettings{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: v1alpha1.SettingsName},
			Spec:       initialSpec(ctx, c, storageClass, pool, log),
		}
		if err := c.Create(ctx, obj); err != nil {
			return err
		}
		log.Info(
			"created panel settings", "storageClass", obj.Spec.DefaultStorageClass, "loadBalancerPool",
			obj.Spec.DefaultLoadBalancerPool,
		)
	} else if err != nil {
		return err
	}
	return migrateServers(ctx, c, obj.Spec, log)
}

func initialSpec(
	ctx context.Context, c client.Client, storageClass, pool string, log *slog.Logger,
) v1alpha1.PanelSettingsSpec {
	var spec v1alpha1.PanelSettingsSpec
	if classes, err := ListStorageClasses(ctx, c); err == nil {
		for _, sc := range classes {
			if sc.Name == storageClass {
				spec.StorageClasses, spec.DefaultStorageClass = []string{sc.Name}, sc.Name
			}
		}
	}
	if spec.DefaultStorageClass == "" {
		log.Warn("initial storage class not found, enable one in the panel settings", "storageClass", storageClass)
	}
	pools, err := ListPools(ctx, c)
	if err != nil {
		log.Warn("load balancer pools unavailable", "err", err)
		return spec
	}
	for _, p := range pools {
		if p.Selectable && (p.Name == pool || p.ServiceLabels["lb.cilium.io/pool"] == pool) {
			spec.LoadBalancerPools, spec.DefaultLoadBalancerPool = []string{p.Name}, p.Name
			break
		}
	}
	return spec
}

// migrateServers fills in storageClass and loadBalancerPool of servers that have none,
// from their existing volume and service (or the defaults).
func migrateServers(ctx context.Context, c client.Client, spec v1alpha1.PanelSettingsSpec, log *slog.Logger) error {
	var list v1alpha1.GameServerList
	if err := c.List(ctx, &list); err != nil {
		return err
	}
	var pools []Pool
	for i := range list.Items {
		gs := &list.Items[i]
		if !tenancy.Owns(gs.Namespace) || (gs.Spec.StorageClass != "" && gs.Spec.LoadBalancerPool != "") {
			continue
		}
		patch := client.MergeFrom(gs.DeepCopy())
		if gs.Spec.StorageClass == "" {
			gs.Spec.StorageClass = cmp.Or(existingStorageClass(ctx, c, gs), spec.DefaultStorageClass)
		}
		if gs.Spec.LoadBalancerPool == "" {
			if pools == nil {
				pools, _ = ListPools(ctx, c)
			}
			gs.Spec.LoadBalancerPool = cmp.Or(existingPool(ctx, c, gs, pools), spec.DefaultLoadBalancerPool)
		}
		if err := c.Patch(ctx, gs, patch); err != nil {
			return err
		}
		log.Info(
			"migrated server settings", "server", gs.Name, "storageClass", gs.Spec.StorageClass, "loadBalancerPool",
			gs.Spec.LoadBalancerPool,
		)
	}
	return nil
}

// existingStorageClass is the storage class of the server's volume ("" when unknown).
func existingStorageClass(ctx context.Context, c client.Client, gs *v1alpha1.GameServer) string {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := c.Get(
		ctx, client.ObjectKey{Namespace: gs.Namespace, Name: gameserver.PVCName(gs.Name)}, pvc,
	); err == nil &&
		pvc.Spec.StorageClassName != nil {
		return *pvc.Spec.StorageClassName
	}
	return ""
}

// existingPool is the pool whose selector matches the server's service ("" when none).
func existingPool(ctx context.Context, c client.Client, gs *v1alpha1.GameServer, pools []Pool) string {
	svc := &corev1.Service{}
	if err := c.Get(
		ctx, client.ObjectKey{Namespace: gs.Namespace, Name: gameserver.ServiceName(gs.Name)}, svc,
	); err != nil {
		return ""
	}
	return poolForLabels(pools, svc.Labels)
}

// poolForLabels returns the selectable pool whose selector matches the labels.
func poolForLabels(pools []Pool, labels map[string]string) string {
	for _, p := range pools {
		if !p.Selectable || len(p.ServiceLabels) == 0 {
			continue
		}
		match := true
		for k, v := range p.ServiceLabels {
			match = match && labels[k] == v
		}
		if match {
			return p.Name
		}
	}
	return ""
}
