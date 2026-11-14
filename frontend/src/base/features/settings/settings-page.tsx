import { useMemo, useState } from "react"
import { SaveIcon, Undo2Icon } from "lucide-react"
import { toast } from "sonner"

import { appSettingsDraft, settingsCards } from "@/app/settings"
import { QueryState } from "@/base/components/query-state"
import { BrandingCard } from "@/base/features/settings/components/branding-card"
import { LegalCard } from "@/base/features/settings/components/legal-card"
import { SecurityCard } from "@/base/features/settings/components/security-card"
import { SsoCard } from "@/base/features/settings/components/sso-card"
import { UpgradeCard } from "@/base/features/settings/components/upgrade-card"
import { VersionsCard } from "@/base/features/settings/components/versions-card"
import { settingsDraft, type SettingsDraft } from "@/base/features/settings/lib/settings-draft"
import { useDraft } from "@/base/hooks/use-draft"
import { PageHeader } from "@/base/layout/page-header"
import { failed, useSettings, useUpdateSettings } from "@/base/lib/queries"
import type { PanelSettings } from "@/base/lib/types"
import { fieldErrors } from "@/base/lib/validation"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"

const header = { title: "Settings" }

/**
 * /settings (admins): branding, legal texts, security (session and token lifetimes, API docs), single sign-on, panel
 * updates and software versions, with the app's settings and cards (app/settings.ts) in between.
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
  const initial = useMemo(() => ({ ...settingsDraft(stored), ...appSettingsDraft(stored) }), [stored])
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
        {settingsCards.top.map((Card, i) => (
          <Card key={i} {...cards} />
        ))}
        <LegalCard {...cards} />
        <SecurityCard {...cards} {...settingsCards.security} />
        <SsoCard {...cards} secretSet={stored.oidcClientSecretSet} />
        {settingsCards.bottom.map((Card, i) => (
          <Card key={i} {...cards} />
        ))}
      </div>
      <UpgradeCard />
      <VersionsCard />
    </div>
  )
}
