package settings

import (
	"context"
	"sort"

	storagev1 "k8s.io/api/storage/v1"
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
		c := StorageClass{
			Name:                 sc.Name,
			Provisioner:          sc.Provisioner,
			ReclaimPolicy:        "Delete",
			VolumeBindingMode:    "Immediate",
			AllowVolumeExpansion: sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion,
			IsDefault:            sc.Annotations[defaultClassAnnotation] == "true",
		}
		if sc.ReclaimPolicy != nil {
			c.ReclaimPolicy = string(*sc.ReclaimPolicy)
		}
		if sc.VolumeBindingMode != nil {
			c.VolumeBindingMode = string(*sc.VolumeBindingMode)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
