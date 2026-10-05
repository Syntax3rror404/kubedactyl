import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { ServerJob } from "@/lib/types"

/** Listing needs the file container, so it waits until that is ready. */
export const useBackups = (server: string, enabled: boolean) =>
  useQuery({
    queryKey: keys.backups(server),
    queryFn: () => api.backups.list(server),
    enabled,
  })

/** Starts a backup job; the job list shows its progress. */
export const useCreateBackup = (server: string, cb?: MutationCallbacks<ServerJob, string>) =>
  useApiMutation(
    (label) => api.backups.create(server, label),
    (qc) => refresh(qc, keys.jobs(server)),
    cb,
  )

/** Starts a restore job. */
export const useRestoreBackup = (server: string, cb?: MutationCallbacks<ServerJob, string>) =>
  useApiMutation(
    (backup) => api.backups.restore(server, backup),
    (qc) => refresh(qc, keys.jobs(server)),
    cb,
  )

export const useDeleteBackup = (server: string, cb?: MutationCallbacks<void, string>) =>
  useApiMutation(
    (backup) => api.backups.delete(server, backup),
    (qc) => refresh(qc, keys.backups(server), keys.allFiles(server)),
    cb,
  )
