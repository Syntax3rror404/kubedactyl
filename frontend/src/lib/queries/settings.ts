import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"

/** The storage classes of the cluster (admins). */
export const useStorageClasses = () =>
  useQuery({ queryKey: keys.storageClasses, queryFn: api.settings.listStorageClasses })

/** Pool details are admin only. */
export const usePools = (enabled = true) => useQuery({ queryKey: keys.pools, queryFn: api.settings.listPools, enabled })
