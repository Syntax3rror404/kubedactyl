import { useMemo, useState } from "react"
import { SaveIcon, Undo2Icon } from "lucide-react"
import { toast } from "sonner"

import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { AddressCard } from "@/features/settings/components/address-card"
import { BrandingCard } from "@/features/settings/components/branding-card"
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
import { settingsDraft, type SettingsDraft } from "@/features/settings/lib/settings-draft"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useSettings, useUpdateSettings } from "@/lib/queries"
import type { PanelSettings } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

const header = { title: "Settings" }

/**
 * /settings (admins): branding, server notice, address, storage classes, pools, egg library, legal texts, security
 * (isolation, session and token lifetimes, subusers per user, API docs), single sign-on, Kube API limit, panel
 * updates and software versions.
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

function SettingsForm({ stored }: { stored: PanelSettings }) {
  const initial = useMemo(() => settingsDraft(stored), [stored])
  const { draft, setDraft, dirty, reset } = useDraft(initial)
  // Discard mounts the cards again, so their own state (a rejected image, typed rows) goes too.
  const [resets, setResets] = useState(0)
  const save = useUpdateSettings({
    onSuccess: () => toast.success("Settings saved"),
    onError: failed("save the settings"),
  })
  const errors = fieldErrors(save.error)
  const discard = () => {
    reset()
    save.reset()
    setResets((n) => n + 1)
  }
  // Functional update: images are read asynchronously, two changes must not overwrite each other.
  const onChange = (patch: Partial<SettingsDraft>) => setDraft((d) => ({ ...d, ...patch }))
  const cards = { draft, onChange, errors }

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
            <Button disabled={!dirty || save.isPending} onClick={() => save.mutate(draft)}>
              {save.isPending ? <Spinner /> : <SaveIcon />}
              Save changes
            </Button>
          </>
        }
      />
      <div key={resets} className="space-y-6">
        <BrandingCard {...cards} />
        <NoticeCard {...cards} />
        <AddressCard {...cards} />
        <StorageClassesCard {...cards} />
        <PoolsCard {...cards} />
        <EggLibrariesCard {...cards} />
        <LegalCard {...cards} />
        <SecurityCard {...cards} />
        <SsoCard {...cards} secretSet={stored.oidcClientSecretSet} />
        <KubeApiCard {...cards} />
      </div>
      <UpgradeCard />
      <VersionsCard />
    </div>
  )
}
