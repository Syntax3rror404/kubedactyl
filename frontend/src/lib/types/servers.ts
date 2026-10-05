// Game servers, their stats, schedules, backups and background jobs.

import type { FilesJob, V1Alpha1GameServer, V1Alpha1Schedule, V1Alpha1ScheduleTask } from "@/lib/types/api.gen"
import type { Stored } from "@/lib/types/common"

export type {
  HttpapiBackup as Backup,
  HttpapiBackupList as BackupList,
  HttpapiCreateServerRequest as CreateServerRequest,
  HttpapiJobList as JobList,
  HttpapiScheduleList as ScheduleList,
  HttpapiScheduleView as ScheduleView,
  HttpapiServerDiagnostics as ServerDiagnostics,
  HttpapiServerStats as ServerStats,
  HttpapiUpdateServerRequest as UpdateServerRequest,
  V1Alpha1Schedule as Schedule,
  V1Alpha1ScheduleTask as ScheduleTask,
  V1Alpha1TrafficPolicy as TrafficPolicy,
} from "@/lib/types/api.gen"

export type GameServer = Stored<V1Alpha1GameServer>
export type GameServerList = { items: GameServer[] }

export type ScheduleAction = V1Alpha1ScheduleTask["action"]
export type ScheduleEvent = NonNullable<V1Alpha1Schedule["event"]>

/** Background job of a server: backup, restore or pull (download from a URL). */
export type ServerJob = FilesJob
