// Package v1alpha1 contains the Kubedactyl API types (Egg, GameServer).
//
// +kubebuilder:object:generate=true
// +groupName=kubedactyl.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is the API group and version of all Kubedactyl resources.
	GroupVersion = schema.GroupVersion{Group: "kubedactyl.io", Version: "v1alpha1"}

	// SchemeBuilder registers the Go types with a runtime.Scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types of this group-version to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	// knownTypes are collected by the init functions of the type files.
	knownTypes []runtime.Object
)

// register adds types to the scheme of this group-version (called from init).
func register(objs ...runtime.Object) { knownTypes = append(knownTypes, objs...) }

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, knownTypes...)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
