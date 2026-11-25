// What this app adds to the shared settings page (base/features/settings): its settings in the form and its cards
// around the base's (branding, legal texts, security, single sign-on, upgrades, versions).

import type { PanelSettings, UpdateSettingsRequest } from "@/base/lib/types"
import { AddressCard } from "@/features/settings/address-card"
import { EggLibrariesCard } from "@/features/settings/egg-libraries-card"
import { KubeApiCard } from "@/features/settings/kube-api-card"
import { NoticeCard } from "@/features/settings/notice-card"
import { PoolsCard } from "@/features/settings/pools-card"
import { IsolationField, SubusersField } from "@/features/settings/security-fields"
import { StorageClassesCard } from "@/features/settings/storage-classes-card"

/** Form values of this app's settings, with every default filled in (see settingsDraft of the base). */
export function appSettingsDraft(s: PanelSettings) {
  return {
    serverNotice: s.serverNotice ?? "",
    externalDomain: s.externalDomain ?? "",
    storageClasses: s.storageClasses ?? [],
    defaultStorageClass: s.defaultStorageClass ?? "",
    loadBalancerPools: s.loadBalancerPools ?? [],
    defaultLoadBalancerPool: s.defaultLoadBalancerPool ?? "",
    allowClusterIP: s.allowClusterIP ?? false,
    eggLibraries: s.eggLibraries ?? [],
    allowPrivateNetworks: s.allowPrivateNetworks ?? false,
    subusersPerUser: s.subusersPerUser ?? 10,
    kubeApiQps: s.kubeApiQps ?? 50,
    kubeApiUserQps: s.kubeApiUserQps ?? 10,
  } satisfies UpdateSettingsRequest
}

/** The app's cards: after the branding, in the security card (around the lifetimes) and after single sign-on. */
export const settingsCards = {
  top: [NoticeCard, AddressCard, StorageClassesCard, PoolsCard, EggLibrariesCard],
  security: {
    description: "Network isolation, lifetimes of sign-ins and API tokens, subusers and the API docs.",
    before: [IsolationField],
    after: [SubusersField],
  },
  bottom: [KubeApiCard],
}
