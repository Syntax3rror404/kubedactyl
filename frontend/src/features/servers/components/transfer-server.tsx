import { useState } from "react"
import { ArrowRightLeftIcon } from "lucide-react"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { DangerRow } from "@/features/servers/components/danger-zone"
import { OwnerSelect } from "@/features/servers/components/owner-select"
import { serverName, serverOwner } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useTransferServer } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Danger zone row: moves a stopped server with its files to another user (admins only). */
export function TransferServer({ server }: { server: GameServer }) {
  const name = server.metadata.name
  const owner = serverOwner(server)
  const [target, setTarget] = useState("")
  const transfer = useTransferServer(name, {
    onSuccess: (_, to) => toast.success(`${serverName(server)} transferred to ${to}`),
    onError: failed("transfer the server"),
  })
  const stopped = server.spec.state !== "Running" && !server.status?.podUID && server.status?.phase !== "Installing"

  return (
    <DangerRow
      title="Transfer server"
      description={
        stopped
          ? "Moves the server with its files and backups to another user. Name and address stay."
          : "Stop the server first to move it to another user."
      }
    >
      <ConfirmDialog
        onOpenChange={() => setTarget("")}
        trigger={
          <Button variant="outline" disabled={!stopped || transfer.isPending}>
            <ArrowRightLeftIcon />
            {transfer.isPending ? "Transferring…" : "Transfer"}
          </Button>
        }
        title={`Transfer ${serverName(server)}?`}
        description="The server and its data move to the new owner. The current owner loses access; the address stays."
        confirmLabel="Transfer"
        disabled={!target || transfer.isPending}
        onConfirm={() => transfer.mutate(target)}
      >
        <Field>
          <FieldLabel htmlFor="transfer-owner">New owner</FieldLabel>
          <OwnerSelect
            id="transfer-owner"
            value={target}
            onChange={setTarget}
            exclude={owner}
            placeholder="Choose a user"
          />
          {owner && <FieldDescription>Current owner: {owner}</FieldDescription>}
        </Field>
      </ConfirmDialog>
    </DangerRow>
  )
}
