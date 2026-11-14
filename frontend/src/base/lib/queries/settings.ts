import { useQuery } from "@tanstack/react-query"

import { settingsDependents } from "@/app/config"
import { baseApi } from "@/base/lib/api"
import { baseKeys } from "@/base/lib/queries/keys"
import { refresh, refreshes, useApiMutation, type MutationCallbacks } from "@/base/lib/queries/mutation"
import type { PanelSettings, UpdateSettingsRequest, UpgradeJob, UpgradeStatus } from "@/base/lib/types"

/** Panel name and version (public; shown in the footer). */
export const usePanelInfo = () =>
  useQuery({ queryKey: baseKeys.info, queryFn: () => baseApi.panel.getInfo(), staleTime: Infinity })

/** The running version, asked every few seconds while the panel restarts after an upgrade. */
export const useLivePanelInfo = () =>
  useQuery({
    queryKey: baseKeys.liveInfo,
    queryFn: () => baseApi.panel.getInfo({ cache: "no-store" }),
    refetchInterval: 3000,
    refetchIntervalInBackground: true,
    retry: false,
    gcTime: 0,
  })

/** Imprint and privacy policy; public, so also used on the sign-in page. Saving the settings refreshes them. */
export const useLegalTexts = () =>
  useQuery({ queryKey: baseKeys.legal, queryFn: baseApi.panel.getLegalTexts, staleTime: Infinity })

/**
 * Name, tagline, logo and favicon of the panel; public, so also used on the sign-in page. Saving the settings
 * refreshes them; a reload revalidates with the ETag (304 while unchanged).
 */
export const useBranding = () =>
  useQuery({ queryKey: baseKeys.branding, queryFn: baseApi.panel.getBranding, staleTime: Infinity })

/** License texts (public); they change only with the panel version. */
export const useLicenses = () =>
  useQuery({ queryKey: baseKeys.licenses, queryFn: baseApi.panel.getLicenses, staleTime: Infinity })

/** Request rates for the footer, every 2 s while the page is visible (these requests do not count). */
export const useRequestRates = () =>
  useQuery({
    queryKey: baseKeys.requestRates,
    queryFn: baseApi.panel.getRequestRates,
    refetchInterval: 2000,
    retry: false,
  })

/** The panel settings; every user reads them, the panel leaves out what only administrators may see. */
export const useSettings = () =>
  useQuery({ queryKey: baseKeys.settings, queryFn: baseApi.settings.get, staleTime: 60_000 })

/** Settings change the legal pages, the branding, single sign-on and what the app names (settingsDependents). */
export const useUpdateSettings = (cb?: MutationCallbacks<PanelSettings, UpdateSettingsRequest>) =>
  useApiMutation(
    baseApi.settings.update,
    (qc, saved) => {
      qc.setQueryData(baseKeys.settings, saved)
      refresh(qc, baseKeys.legal, baseKeys.branding, baseKeys.oidcSignIn, ...settingsDependents)
    },
    cb,
  )

/** Versions of the panel, Go and the main Go modules (admins). */
export const useVersions = () =>
  useQuery({ queryKey: baseKeys.versions, queryFn: baseApi.panel.getVersions, staleTime: Infinity })

/** Panel updates (admins); the registry itself is only checked every few minutes by the panel. */
export const useUpgradeStatus = (enabled = true) =>
  useQuery({
    queryKey: baseKeys.upgrade,
    queryFn: () => baseApi.upgrade.getStatus(),
    enabled,
    refetchInterval: (q) => (q.state.data?.jobs.some((j) => j.state === "running") ? 3000 : 60_000),
  })

/** Asks the registry now instead of waiting for the next check. */
export const useRefreshUpgradeStatus = (cb?: MutationCallbacks<UpgradeStatus, void>) =>
  useApiMutation(
    () => baseApi.upgrade.getStatus(true),
    (qc, st) => qc.setQueryData(baseKeys.upgrade, st),
    cb,
  )

/** Starts the upgrade job to a chart version. */
export const useStartUpgrade = (cb?: MutationCallbacks<UpgradeJob, string>) =>
  useApiMutation(baseApi.upgrade.start, refreshes(baseKeys.upgrade), cb)
