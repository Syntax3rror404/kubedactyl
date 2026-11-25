// Shapes of the Kubedactyl API (mirrors backend/api/v1alpha1 and internal/httpapi). The types themselves are
// generated from the Go structs (lib/api.gen.ts by swagger-typescript-api, `make docs`); this file gives them the
// names the frontend uses and refines where the Go types cannot say more. The base API's shapes (users, settings,
// upgrades) are in base/lib/types.ts.

import type {
  FilesJob,
  HttpapiPowerRequest,
  V1Alpha1Egg,
  V1Alpha1GameServer,
  V1Alpha1Schedule,
  V1Alpha1ScheduleTask,
  V1ObjectMeta,
} from "@/lib/api.gen"

// Shared by several areas.

export type { ChecksStatus as CheckStatus, V1Alpha1Phase as Phase } from "@/lib/api.gen"

/**
 * Metadata of an object read from the cluster: the API server always sets name, namespace, uid and
 * the creation time (optional in the Kubernetes Go type, because it is also used for new objects).
 */
type ObjectMeta = V1ObjectMeta & Required<Pick<V1ObjectMeta, "name" | "namespace" | "uid" | "creationTimestamp">>

/** A cluster object as the panel returns it: with complete metadata. */
type Stored<T extends { metadata?: V1ObjectMeta }> = Omit<T, "metadata"> & { metadata: ObjectMeta }

export type PowerSignal = HttpapiPowerRequest["signal"]

// Eggs: game server templates (Pterodactyl/Pelican).

export type {
  EggstoreLibraryEgg as LibraryEgg,
  EggstoreLibraryRepository as LibraryRepository,
  HttpapiLibraryEgg as LibraryEggContent,
  HttpapiLibraryList as LibraryList,
  HttpapiLibraryRepositoryCheck as LibraryRepositoryCheck,
  V1Alpha1ConfigFile as ConfigFile,
  V1Alpha1ConfigReplace as ConfigReplace,
  V1Alpha1EggSpec as EggSpec,
  V1Alpha1EggVariable as EggVariable,
} from "@/lib/api.gen"

export type Egg = Stored<V1Alpha1Egg>
export type EggList = { items: Egg[] }

/** Formats of GET /eggs/{egg}/export (query parameter). */
export type EggExportFormat = "yaml" | "json" | "ptdl"

// Game servers, their stats, schedules, backups and background jobs.

export type {
  HttpapiBackup as Backup,
  HttpapiBackupList as BackupList,
  HttpapiCreateServerRequest as CreateServerRequest,
  HttpapiJobList as JobList,
  HttpapiMigrateRequest as MigrateRequest,
  HttpapiScheduleList as ScheduleList,
  HttpapiScheduleView as ScheduleView,
  HttpapiServerDiagnostics as ServerDiagnostics,
  HttpapiServerStats as ServerStats,
  HttpapiUpdateServerRequest as UpdateServerRequest,
  V1Alpha1Schedule as Schedule,
  V1Alpha1ShareAccess as ShareAccess,
  V1Alpha1ScheduleTask as ScheduleTask,
  V1Alpha1TrafficPolicy as TrafficPolicy,
} from "@/lib/api.gen"

export type GameServer = Stored<V1Alpha1GameServer>
export type GameServerList = { items: GameServer[] }

export type ScheduleAction = V1Alpha1ScheduleTask["action"]
export type ScheduleEvent = NonNullable<V1Alpha1Schedule["event"]>

/** Background job of a server: backup, restore, download from a URL, compression or extraction. */
export type ServerJob = FilesJob

// File manager.

export type {
  FilesEntry as FileEntry,
  HttpapiFileList as DirectoryListing,
  HttpapiFilesSession as FilesSession,
} from "@/lib/api.gen"

// Cluster page and health checks.

export type {
  ClusterinfoCertInfo as CertInfo,
  ClusterinfoHealth as ClusterHealth,
  ClusterinfoIdentity as ClusterIdentity,
  ClusterinfoNode as ClusterNode,
  ClusterinfoNodesResponse as NodesResponse,
  ClusterinfoTotals as ClusterTotals,
  HttpapiClusterInfo as ClusterInfo,
} from "@/lib/api.gen"

// Panel settings of this app: storage classes and load balancer pools to pick from.

export type {
  HttpapiPoolList as PoolList,
  HttpapiStorageClassList as StorageClassList,
  AppsettingsPool as Pool,
} from "@/lib/api.gen"
