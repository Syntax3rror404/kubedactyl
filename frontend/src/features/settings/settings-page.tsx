import { SaveIcon, Undo2Icon } from "lucide-react"
import { toast } from "sonner"

import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { AddressCard } from "@/features/settings/components/address-card"
import { BrandingCard } from "@/features/settings/components/branding-card"
import type { Selection, SelectionHandlers } from "@/features/settings/components/default-cell"
import { EggLibrariesCard } from "@/features/settings/components/egg-libraries-card"
import { KubeApiCard } from "@/features/settings/components/kube-api-card"
import { LegalCard } from "@/features/settings/components/legal-card"
import { NoticeCard } from "@/features/settings/components/notice-card"
import { PoolsCard } from "@/features/settings/components/pools-card"
import { SecurityCard } from "@/features/settings/components/security-card"
import { SsoCard } from "@/features/settings/components/sso-card"
import { StorageClassesCard } from "@/features/settings/components/storage-classes-card"
import { UpgradeCard } from "@/features/settings/components/upgrade-card"
import { VersionsCard } from "@/features/settings/components/versions-card"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { usePools, useUpdateSettings, useServers, useSettings, useStorageClasses } from "@/lib/queries"
import type { PanelSettings, UpdateSettingsRequest } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

const header = { title: "Settings" }

/**
 * /settings (admins): branding, server notice, address, storage classes, pools, egg library, legal texts, security
 * (isolation, session and token lifetimes, API docs), single sign-on, Kube API limit, panel updates and software
 * versions.
 */
export function PanelSettingsPage() {
  const settings = useSettings()
  return (
    <QueryState
      query={settings}
      skeleton={
        <div className="space-y-6">
          <PageHeader {...header} />
          <Skeleton className="h-[60vh] rounded-2xl" />
        </div>
      }
    >
      {(stored) => <SettingsForm key={JSON.stringify(stored)} stored={stored} />}
    </QueryState>
  )
}

/** The stored settings plus a client secret typed for single sign-on (sent only when one was typed). */
type Draft = PanelSettings & Pick<UpdateSettingsRequest, "oidcClientSecret">

function SettingsForm({ stored }: { stored: PanelSettings }) {
  const classes = useStorageClasses()
  const pools = usePools()
  const servers = useServers()
  const { draft: value, setDraft, dirty, reset } = useDraft<Draft>(stored)
  const save = useUpdateSettings({
    onSuccess: () => toast.success("Settings saved"),
    onError: failed("save the settings"),
  })
  const errors = fieldErrors(save.error)
  const discard = () => {
    reset()
    save.reset()
  }

  // Functional update: images are read asynchronously, two changes must not overwrite each other.
  const change = (patch: Partial<Draft>) => setDraft((d) => ({ ...d, ...patch }))
  const storage: Selection = { enabled: value.storageClasses ?? [], defaultName: value.defaultStorageClass ?? "" }
  const pool: Selection = { enabled: value.loadBalancerPools ?? [], defaultName: value.defaultLoadBalancerPool ?? "" }
  const exampleIP = servers.data?.find((s) => s.status?.address && s.spec.serviceType !== "ClusterIP")?.status?.address

  return (
    <div className="space-y-6">
      <PageHeader
        {...header}
        actions={
          <>
            {dirty && (
              <Button variant="ghost" onClick={discard}>
                <Undo2Icon />
                Discard
              </Button>
            )}
            <Button
              disabled={!dirty || save.isPending}
              onClick={() => save.mutate({ ...value, oidcClientSecret: value.oidcClientSecret || undefined })}
            >
              {save.isPending ? <Spinner /> : <SaveIcon />}
              Save changes
            </Button>
          </>
        }
      />

      <BrandingCard
        values={{
          brandName: value.brandName ?? "",
          brandTagline: value.brandTagline ?? "",
          brandLogo: value.brandLogo ?? "",
          favicon: value.favicon ?? "",
        }}
        onChange={change}
        errors={errors}
      />

      <NoticeCard
        notice={value.serverNotice ?? ""}
        onChange={(serverNotice) => change({ serverNotice })}
        error={errors.serverNotice}
      />

      <AddressCard
        domain={value.externalDomain ?? ""}
        onChange={(externalDomain) => change({ externalDomain })}
        exampleIP={exampleIP}
        error={errors.externalDomain}
      />

      <StorageClassesCard
        classes={classes.data}
        selection={storage}
        fieldError={errors.storageClasses}
        {...handlers(storage, (s) => change({ storageClasses: s.enabled, defaultStorageClass: s.defaultName }))}
      />

      <PoolsCard
        pools={pools.data?.items}
        ipv6Missing={pools.data?.ipv6Missing}
        loadError={pools.data?.error ?? pools.error?.message}
        fieldError={errors.loadBalancerPools}
        selection={pool}
        allowClusterIP={value.allowClusterIP ?? false}
        onAllowClusterIP={(allowClusterIP) => change({ allowClusterIP })}
        {...handlers(pool, (s) => change({ loadBalancerPools: s.enabled, defaultLoadBalancerPool: s.defaultName }))}
      />

      <EggLibrariesCard
        repositories={value.eggLibraries ?? []}
        onChange={(eggLibraries) => change({ eggLibraries })}
        error={errors.eggLibraries}
      />

      <LegalCard
        values={{ legalNotice: value.legalNotice ?? "", privacyPolicy: value.privacyPolicy ?? "" }}
        onChange={(key, text) => change({ [key]: text })}
        errors={errors}
      />

      <SecurityCard
        values={{
          allowPrivateNetworks: value.allowPrivateNetworks ?? false,
          sessionHours: value.sessionHours ?? 0,
          apiTokenMaxDays: value.apiTokenMaxDays ?? 0,
          disableApiDocs: value.disableApiDocs ?? false,
        }}
        onChange={change}
        errors={errors}
      />

      <SsoCard
        values={value.oidc ?? {}}
        onChange={(patch) => setDraft((d) => ({ ...d, oidc: { ...d.oidc, ...patch } }))}
        secret={value.oidcClientSecret ?? ""}
        secretSet={value.oidcClientSecretSet}
        onSecret={(oidcClientSecret) => change({ oidcClientSecret })}
        errors={errors}
      />

      <KubeApiCard
        values={{ kubeApiQps: value.kubeApiQps ?? 50, kubeApiUserQps: value.kubeApiUserQps ?? 10 }}
        onChange={change}
        errors={errors}
      />

      <UpgradeCard />

      <VersionsCard />
    </div>
  )
}

/** Enabling adds a name, disabling removes it; the default moves to the first enabled entry when needed. */
function handlers(sel: Selection, apply: (sel: Selection) => void): SelectionHandlers {
  return {
    onToggle: (name, on) => {
      const enabled = on ? [...sel.enabled.filter((n) => n !== name), name] : sel.enabled.filter((n) => n !== name)
      apply({ enabled, defaultName: enabled.includes(sel.defaultName) ? sel.defaultName : (enabled[0] ?? "") })
    },
    onDefault: (name) => apply({ ...sel, defaultName: name }),
  }
}
