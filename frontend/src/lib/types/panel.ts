// The panel itself: name and version, legal texts, settings and self-upgrades.

export type {
  HttpapiBranding as Branding,
  HttpapiLegalTexts as LegalTexts,
  HttpapiPoolList as PoolList,
  HttpapiRequestRates as RequestRates,
  HttpapiSettingsView as PanelSettings,
  HttpapiStorageClassList as StorageClassList,
  HttpapiUpdateSettingsRequest as UpdateSettingsRequest,
  HttpapiUpgradeStatus as UpgradeStatus,
  HttpapiVersions as Versions,
  HttpserverInfoResponse as PanelInfo,
  KubeRate as Rate,
  SelfupgradeJob as UpgradeJob,
  SettingsPool as Pool,
  SettingsStorageClass as StorageClass,
  V1Alpha1OIDCSettings as OIDCSettings,
} from "@/lib/types/api.gen"
