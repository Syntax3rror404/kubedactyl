// Package crd embeds the generated CustomResourceDefinitions (go tool controller-gen).
package crd

import "embed"

// Files contains the CRD manifests.
//
//go:embed *.yaml
var Files embed.FS
