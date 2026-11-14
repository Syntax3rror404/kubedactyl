import { memberGroup } from "@/app/config"
import type { appSettingsDraft } from "@/app/settings"
import type { PanelSettings, UpdateSettingsRequest } from "@/base/lib/types"

/**
 * Form values of the base's settings (branding, legal texts, security, single sign-on): the stored settings with
 * every default filled in once, so typing a value and erasing it again leaves the form unchanged; the app adds its
 * own (appSettingsDraft in app/settings.ts). `oidcClientSecret`: undefined keeps the stored secret, "" removes it.
 */
export function settingsDraft(s: PanelSettings) {
  const o = s.oidc ?? {}
  return {
    brandName: s.brandName ?? "",
    brandTagline: s.brandTagline ?? "",
    brandLogo: s.brandLogo ?? "",
    favicon: s.favicon ?? "",
    legalNotice: s.legalNotice ?? "",
    privacyPolicy: s.privacyPolicy ?? "",
    sessionHours: s.sessionHours ?? 12,
    apiTokenMaxDays: s.apiTokenMaxDays ?? 90,
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
      [memberGroup]: o[memberGroup] ?? "",
      linkByUsername: o.linkByUsername ?? false,
      keepPasswords: o.keepPasswords ?? false,
    },
    oidcClientSecret: undefined as string | undefined,
  } satisfies UpdateSettingsRequest
}

/** The whole form: the base's settings and the app's. */
export type SettingsDraft = ReturnType<typeof settingsDraft> & ReturnType<typeof appSettingsDraft>

/** What every card of the settings page gets: the draft, its change handler and the errors of the last save. */
export interface SettingsCardProps {
  draft: SettingsDraft
  onChange: (patch: Partial<SettingsDraft>) => void
  errors: Record<string, string>
}

/** The error of a text over its length limit (characters), checked before saving. */
export const tooLong = (text: string, max: number) =>
  text.length > max ? `Too long (${text.length} of ${max} characters)` : undefined
