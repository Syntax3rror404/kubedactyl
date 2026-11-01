import type { PanelSettings, UpdateSettingsRequest } from "@/lib/types"

/**
 * Form values of the settings page: the stored settings with every default filled in once, so typing a value and
 * erasing it again leaves the form unchanged. `oidcClientSecret`: undefined keeps the stored secret, "" removes it.
 */
export function settingsDraft(s: PanelSettings) {
  const o = s.oidc ?? {}
  return {
    brandName: s.brandName ?? "",
    brandTagline: s.brandTagline ?? "",
    brandLogo: s.brandLogo ?? "",
    favicon: s.favicon ?? "",
    serverNotice: s.serverNotice ?? "",
    externalDomain: s.externalDomain ?? "",
    storageClasses: s.storageClasses ?? [],
    defaultStorageClass: s.defaultStorageClass ?? "",
    loadBalancerPools: s.loadBalancerPools ?? [],
    defaultLoadBalancerPool: s.defaultLoadBalancerPool ?? "",
    allowClusterIP: s.allowClusterIP ?? false,
    eggLibraries: s.eggLibraries ?? [],
    legalNotice: s.legalNotice ?? "",
    privacyPolicy: s.privacyPolicy ?? "",
    allowPrivateNetworks: s.allowPrivateNetworks ?? false,
    sessionHours: s.sessionHours ?? 12,
    apiTokenMaxDays: s.apiTokenMaxDays ?? 90,
    subusersPerUser: s.subusersPerUser ?? 10,
    disableApiDocs: s.disableApiDocs ?? false,
    oidc: {
      enabled: o.enabled ?? false,
      name: o.name ?? "",
      issuerUrl: o.issuerUrl ?? "",
      clientId: o.clientId ?? "",
      redirectUrl: o.redirectUrl ?? "",
      usernameClaim: o.usernameClaim ?? "",
      groupsClaim: o.groupsClaim ?? "",
      adminGroup: o.adminGroup ?? "",
      userGroup: o.userGroup ?? "",
      linkByUsername: o.linkByUsername ?? false,
      keepPasswords: o.keepPasswords ?? false,
    },
    oidcClientSecret: undefined as string | undefined,
    kubeApiQps: s.kubeApiQps ?? 50,
    kubeApiUserQps: s.kubeApiUserQps ?? 10,
  } satisfies UpdateSettingsRequest
}

export type SettingsDraft = ReturnType<typeof settingsDraft>

/** What every card of the settings page gets: the draft, its change handler and the errors of the last save. */
export interface SettingsCardProps {
  draft: SettingsDraft
  onChange: (patch: Partial<SettingsDraft>) => void
  errors: Record<string, string>
}

/** A list of enabled names plus one default, e.g. storage classes or load balancer pools. */
export interface Selection {
  enabled: string[]
  defaultName: string
}

/** Callbacks of a selection table. */
export interface SelectionHandlers {
  onToggle: (name: string, on: boolean) => void
  onDefault: (name: string) => void
}

/** Enabling adds a name, disabling removes it; the default moves to the first enabled entry when needed. */
export function selectionHandlers(sel: Selection, apply: (sel: Selection) => void): SelectionHandlers {
  return {
    onToggle: (name, on) => {
      const enabled = on ? [...sel.enabled.filter((n) => n !== name), name] : sel.enabled.filter((n) => n !== name)
      apply({ enabled, defaultName: enabled.includes(sel.defaultName) ? sel.defaultName : (enabled[0] ?? "") })
    },
    onDefault: (name) => apply({ ...sel, defaultName: name }),
  }
}

/** The error of a text over its length limit (characters), checked before saving. */
export const tooLong = (text: string, max: number) =>
  text.length > max ? `Too long (${text.length} of ${max} characters)` : undefined
