import { useState } from "react"
import { ForkliftIcon } from "lucide-react"
import { useNavigate } from "react-router"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"
import { DangerRow } from "@/features/servers/components/danger-zone"
import { PlacementSelect } from "@/features/servers/components/placement-fields"
import { serverName } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useMigrateServer, useSettings } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Danger zone row: moves the server files to a volume of another storage class (admins only). */
export function MigrateServer({ server }: { server: GameServer }) {
  const name = server.metadata.name
  const navigate = useNavigate()
  const settings = useSettings()
  const current = server.status?.storageClass || server.spec.storageClass || ""
  const options = (settings.data?.storageClasses ?? []).filter((c) => c !== current)
  const [target, setTarget] = useState("")
  const [start, setStart] = useState(false)
  const migrate = useMigrateServer(name, {
    onSuccess: () => {
      toast.success("Storage migration started")
      navigate(`/servers/${name}`)
    },
    onError: failed("start the storage migration"),
  })
  const failure = server.status?.migration?.error

  return (
    <DangerRow
      title="Migrate storage class"
      description={
        failure
          ? `The last migration failed: ${failure}`
          : options.length
            ? "Moves everything incl. backups to a new volume of the target storage class."
            : "No storage class available to migrate."
      }
    >
      <ConfirmDialog
        onOpenChange={() => {
          setTarget("")
          setStart(false)
        }}
        trigger={
          <Button variant="outline" disabled={!options.length || migrate.isPending}>
            <ForkliftIcon />
            Migrate
          </Button>
        }
        title={`Migrate ${serverName(server)}?`}
        description="The server stops and stays locked until its files are copied and checked. The old volume is deleted then."
        confirmLabel="Migrate"
        disabled={!target || migrate.isPending}
        onConfirm={() => migrate.mutate({ storageClass: target, startOnCompletion: start })}
      >
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="migrate-storage">Storage class</FieldLabel>
            <PlacementSelect
              id="migrate-storage"
              value={target}
              options={options}
              defaultName={settings.data?.defaultStorageClass}
              placeholder="Choose a storage class"
              onChange={setTarget}
            />
            {current && <FieldDescription>Current storage class: {current}</FieldDescription>}
          </Field>
          <Field orientation="horizontal">
            <Switch id="migrate-start" checked={start} onCheckedChange={setStart} />
            <FieldLabel htmlFor="migrate-start">Start the server after the migration</FieldLabel>
          </Field>
        </FieldGroup>
      </ConfirmDialog>
    </DangerRow>
  )
}
