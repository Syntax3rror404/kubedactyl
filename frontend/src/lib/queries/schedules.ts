import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { Schedule, ScheduleList } from "@/lib/types"

/** Schedules of a server; refreshed so "running" and the last result stay current. */
export const useSchedules = (server: string) =>
  useQuery({
    queryKey: keys.schedules(server),
    queryFn: () => api.schedules.list(server),
    refetchInterval: 10_000,
  })

/** Saves the whole list of schedules. */
export const useUpdateSchedules = (server: string, cb?: MutationCallbacks<ScheduleList, Schedule[]>) =>
  useApiMutation(
    (items) => api.schedules.update(server, items),
    (qc, list) => qc.setQueryData(keys.schedules(server), list),
    cb,
  )

export const useRunSchedule = (server: string, cb?: MutationCallbacks<void, string>) =>
  useApiMutation(
    (schedule) => api.schedules.run(server, schedule),
    (qc) => refresh(qc, keys.schedules(server)),
    cb,
  )
