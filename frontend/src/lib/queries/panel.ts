import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"

import { api, setThrottledHandler, type Throttle } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type { PanelSettings, UpgradeJob, UpgradeStatus } from "@/lib/types"

/** Panel name and version (public; shown in the footer). */
export const usePanelInfo = () =>
  useQuery({ queryKey: keys.info, queryFn: () => api.panel.getInfo(), staleTime: Infinity })

/** The running version, asked every few seconds while the panel restarts after an upgrade. */
export const useLivePanelInfo = () =>
  useQuery({
    queryKey: keys.liveInfo,
    queryFn: () => api.panel.getInfo({ cache: "no-store" }),
    refetchInterval: 3000,
    refetchIntervalInBackground: true,
    retry: false,
    gcTime: 0,
  })

/** Imprint and privacy policy; public, so also used on the sign-in page. */
export const useLegalTexts = () =>
  useQuery({ queryKey: keys.legal, queryFn: api.panel.getLegalTexts, staleTime: 5 * 60_000 })

/** Name, tagline, logo and favicon of the panel; public, so also used on the sign-in page. */
export const useBranding = () =>
  useQuery({ queryKey: keys.branding, queryFn: api.panel.getBranding, staleTime: 5 * 60_000 })

/** License texts (public); they change only with the panel version. */
export const useLicenses = () =>
  useQuery({ queryKey: keys.licenses, queryFn: api.panel.getLicenses, staleTime: Infinity })

/** Request rates for the footer, every 2 s while the page is visible (these requests do not count). */
export const useRequestRates = () =>
  useQuery({ queryKey: keys.requestRates, queryFn: api.panel.getRequestRates, refetchInterval: 2000, retry: false })

/** The last request the panel refused for now (too many requests, panel busy) until it may be sent again. */
export function useThrottle() {
  const [throttle, setThrottle] = useState<(Throttle & { until: number }) | null>(null)
  useEffect(() => {
    // Shown for at least 5 s, so it does not flicker between the polls of a page.
    setThrottledHandler((t) => setThrottle({ ...t, until: Date.now() + Math.max(t.seconds, 5) * 1000 }))
    return () => setThrottledHandler(null)
  }, [])
  useEffect(() => {
    if (!throttle) return
    const id = setTimeout(() => setThrottle(null), throttle.until - Date.now())
    return () => clearTimeout(id)
  }, [throttle])
  return throttle
}

export const useSettings = () => useQuery({ queryKey: keys.settings, queryFn: api.settings.get, staleTime: 60_000 })

export const useStorageClasses = () =>
  useQuery({ queryKey: keys.storageClasses, queryFn: api.settings.listStorageClasses })

/** Pool details are admin only. */
export const usePools = (enabled = true) => useQuery({ queryKey: keys.pools, queryFn: api.settings.listPools, enabled })

/**
 * Settings change what the cluster card, servers (address), the legal pages, the branding and the egg library
 * (repositories) show.
 */
export const useUpdateSettings = (cb?: MutationCallbacks<PanelSettings, PanelSettings>) =>
  useApiMutation(
    api.settings.update,
    (qc, saved) => {
      qc.setQueryData(keys.settings, saved)
      refresh(qc, keys.cluster, keys.servers, keys.legal, keys.branding, keys.library)
    },
    cb,
  )

/** Versions of the panel, Go and the main Go modules (admins). */
export const useVersions = () =>
  useQuery({ queryKey: keys.versions, queryFn: api.panel.getVersions, staleTime: Infinity })

/** Panel updates (admins); the registry itself is only checked every few minutes by the panel. */
export const useUpgradeStatus = (enabled = true) =>
  useQuery({
    queryKey: keys.upgrade,
    queryFn: () => api.upgrade.getStatus(),
    enabled,
    refetchInterval: (q) => (q.state.data?.jobs.some((j) => j.state === "running") ? 3000 : 60_000),
  })

/** Asks the registry now instead of waiting for the next check. */
export const useRefreshUpgradeStatus = (cb?: MutationCallbacks<UpgradeStatus, void>) =>
  useApiMutation(
    () => api.upgrade.getStatus(true),
    (qc, st) => qc.setQueryData(keys.upgrade, st),
    cb,
  )

export const useStartUpgrade = (cb?: MutationCallbacks<UpgradeJob, string>) =>
  useApiMutation(api.upgrade.start, (qc) => refresh(qc, keys.upgrade), cb)
