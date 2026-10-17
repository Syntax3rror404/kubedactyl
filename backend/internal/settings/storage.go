package settings

import (
	"context"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StorageClass describes a storage class of the cluster.
type StorageClass struct {
	Name                 string `json:"name"                 example:"longhorn"`
	Provisioner          string `json:"provisioner"          example:"driver.longhorn.io"`
	ReclaimPolicy        string `json:"reclaimPolicy"        example:"Retain"`
	VolumeBindingMode    string `json:"volumeBindingMode"    example:"Immediate"`
	AllowVolumeExpansion bool   `json:"allowVolumeExpansion"`
	// IsDefault is true for the cluster default storage class.
	IsDefault bool `json:"isDefault"`
}

const defaultClassAnnotation = "storageclass.kubernetes.io/is-default-class"

// ListStorageClasses returns all storage classes of the cluster, sorted by name.
func ListStorageClasses(ctx context.Context, r client.Reader) ([]StorageClass, error) {
	var list storagev1.StorageClassList
	if err := r.List(ctx, &list); err != nil {
		return nil, err
	}
	out := make([]StorageClass, 0, len(list.Items))
	for _, sc := range list.Items {
		out = append(out, StorageClass{
			Name:                 sc.Name,
			Provisioner:          sc.Provisioner,
			ReclaimPolicy:        string(ptr.Deref(sc.ReclaimPolicy, corev1.PersistentVolumeReclaimDelete)),
			VolumeBindingMode:    string(ptr.Deref(sc.VolumeBindingMode, storagev1.VolumeBindingImmediate)),
			AllowVolumeExpansion: ptr.Deref(sc.AllowVolumeExpansion, false),
			IsDefault:            sc.Annotations[defaultClassAnnotation] == "true",
		})
	}
	slices.SortFunc(out, func(a, b StorageClass) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}
