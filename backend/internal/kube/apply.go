package kube

import (
	"context"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrPatch works like controllerutil.CreateOrPatch, but reads the current object with
// reader, which should be uncached. Behind API proxies that cut watches the informer cache
// can lag behind by minutes; deciding "create or patch" from it fails with AlreadyExists
// for objects that were created a moment ago.
func CreateOrPatch(
	ctx context.Context,
	reader client.Reader,
	c client.Client,
	obj client.Object,
	mutate func() error,
) error {
	if err := reader.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := mutate(); err != nil {
			return err
		}
		return c.Create(ctx, obj)
	}
	before := obj.DeepCopyObject().(client.Object)
	if err := mutate(); err != nil {
		return err
	}
	if equality.Semantic.DeepEqual(before, obj) {
		return nil
	}
	return c.Patch(ctx, obj, client.MergeFrom(before))
}
