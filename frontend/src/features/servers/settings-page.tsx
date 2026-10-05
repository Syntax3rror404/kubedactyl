import { SaveIcon } from "lucide-react"
import { useOutletContext } from "react-router"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { DangerZone } from "@/features/servers/components/danger-zone"
import { GeneralCard } from "@/features/servers/components/general-card"
import { KubernetesCard } from "@/features/servers/components/kubernetes-card"
import { NetworkCard } from "@/features/servers/components/network-card"
import { ResourcesCard, ResourceSummary } from "@/features/servers/components/resources-card"
import { settingsChanges, settingsDraft } from "@/features/servers/lib/settings-draft"
import type { ServerContext } from "@/features/servers/server-layout"
import { useAuth } from "@/hooks/use-auth"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useSettings, useUpdateServer } from "@/lib/queries"
import { fieldErrors } from "@/lib/validation"

/**
 * /servers/:server/settings: name, resources, ports, pool, fixed IP and traffic policy; danger zone
 * (reinstall, suspend, transfer, delete).
 */
export function ServerSettingsPage() {
  const { server } = useOutletContext<ServerContext>()
  const { isAdmin } = useAuth()
  const settings = useSettings()
  const { draft, setDraft, dirty } = useDraft(settingsDraft(server))
  const onChange = (patch: Partial<typeof draft>) => setDraft((d) => ({ ...d, ...patch }))
  const save = useUpdateServer(server.metadata.name, {
    onSuccess: () => {
      toast.success("Settings saved", {
        description: "Network and disk changes apply now, CPU and memory on the next start.",
      })
    },
    onError: failed("save the settings"),
  })
  const errors = fieldErrors(save.error)
  const cards = { server, draft, onChange, errors }

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <GeneralCard {...cards} isAdmin={isAdmin} />
      {isAdmin ? (
        <ResourcesCard {...cards} />
      ) : (
        <ResourceSummary server={server} domain={settings.data?.externalDomain} />
      )}
      <NetworkCard {...cards} isAdmin={isAdmin} />
      {isAdmin && <KubernetesCard server={server} />}

      <div className="flex justify-end lg:col-span-2">
        <Button
          size="lg"
          disabled={!dirty || save.isPending || (isAdmin && draft.ports.length === 0)}
          onClick={() => save.mutate(settingsChanges(server, draft, isAdmin))}
        >
          {save.isPending ? <Spinner /> : <SaveIcon />}
          Save settings
        </Button>
      </div>

      <DangerZone server={server} className="lg:col-span-2" isAdmin={isAdmin} />
    </div>
  )
}
