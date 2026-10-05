// Types shared by several API areas. The API types themselves are generated from the Go structs
// (api.gen.ts by swagger-typescript-api, `make docs`); the files in this folder give them the names the
// frontend uses and refine where the Go types cannot say more.

import type { HttpapiPowerRequest, V1ObjectMeta } from "@/lib/types/api.gen"

export type { ChecksStatus as CheckStatus, V1Alpha1Phase as Phase } from "@/lib/types/api.gen"

/**
 * Metadata of an object read from the cluster: the API server always sets name, namespace, uid and
 * the creation time (optional in the Kubernetes Go type, because it is also used for new objects).
 */
type ObjectMeta = V1ObjectMeta & Required<Pick<V1ObjectMeta, "name" | "namespace" | "uid" | "creationTimestamp">>

/** A cluster object as the panel returns it: with complete metadata. */
export type Stored<T extends { metadata?: V1ObjectMeta }> = Omit<T, "metadata"> & { metadata: ObjectMeta }

export type PowerSignal = HttpapiPowerRequest["signal"]
