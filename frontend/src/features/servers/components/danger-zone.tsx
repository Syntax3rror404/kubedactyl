import { useState } from "react"
import { LockIcon, LockOpenIcon, RefreshCcwIcon, Trash2Icon } from "lucide-react"
import { useNavigate } from "react-router"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { MigrateServer } from "@/features/servers/components/migrate-server"
import { TransferServer } from "@/features/servers/components/transfer-server"
import { suspendFeedback } from "@/features/servers/lib/feedback"
import { serverName } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useDeleteServer, useReinstallServer, useSuspendServer } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/** Reinstall, and for admins suspend, transfer, storage migration and delete, all behind a confirmation. */
export function DangerZone({
  server,
  className,
  isAdmin = false,
}: {
  server: GameServer
  className?: string
  isAdmin?: boolean
}) {
  const name = server.metadata.name
  const navigate = useNavigate()
  const [confirm, setConfirm] = useState("")
  const suspend = useSuspendServer(name, suspendFeedback)
  const suspended = !!server.spec.suspended

  const reinstall = useReinstallServer(name, {
    onSuccess: () => {
      toast.success("Reinstall started")
      navigate(`/servers/${name}`)
    },
    onError: failed("start the reinstall"),
  })
  const remove = useDeleteServer(name, {
    onSuccess: () => {
      toast.success(`${serverName(server)} deleted`)
      navigate("/")
    },
    onError: failed("delete the server"),
  })

  return (
    <Card className={`border-destructive/40 ${className ?? ""}`}>
      <CardHeader>
        <CardTitle className="text-destructive">Danger zone</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-2">
        <DangerRow
          title="Reinstall server"
          description="Stops the server and runs the install script again. Files are kept unless the script removes them."
        >
          <ConfirmDialog
            trigger={
              <Button variant="outline">
                <RefreshCcwIcon />
                Reinstall
              </Button>
            }
            title={`Reinstall ${serverName(server)}?`}
            description="The server is stopped and the egg install script runs again."
            confirmLabel="Reinstall"
            onConfirm={() => reinstall.mutate()}
          />
        </DangerRow>
        {isAdmin && (
          <DangerRow
            title={suspended ? "Unsuspend server" : "Suspend server"}
            description={
              suspended
                ? "The owner can start and use the server again."
                : "Stops the server and locks it for its owner. Files are kept; admins keep access."
            }
          >
            {suspended ? (
              <Button variant="outline" disabled={suspend.isPending} onClick={() => suspend.mutate(false)}>
                <LockOpenIcon />
                Unsuspend
              </Button>
            ) : (
              <ConfirmDialog
                trigger={
                  <Button variant="outline">
                    <LockIcon />
                    Suspend
                  </Button>
                }
                title={`Suspend ${serverName(server)}?`}
                description="The server stops. Its owner can still see it but no longer use it."
                confirmLabel="Suspend"
                destructive
                onConfirm={() => suspend.mutate(true)}
              />
            )}
          </DangerRow>
        )}
        {isAdmin && <TransferServer server={server} />}
        {isAdmin && <MigrateServer server={server} />}
        {isAdmin && (
          <DangerRow
            title="Delete server"
            description="Removes the server, its files and the data volume permanently."
            destructive
          >
            <ConfirmDialog
              onOpenChange={() => setConfirm("")}
              trigger={
                <Button variant="destructive">
                  <Trash2Icon />
                  Delete
                </Button>
              }
              title={`Delete ${serverName(server)}?`}
              description={
                <>
                  All files including the volume are deleted. Type <strong className="font-mono">{name}</strong> to
                  confirm.
                </>
              }
              confirmLabel="Delete permanently"
              destructive
              disabled={confirm !== name || remove.isPending}
              onConfirm={() => remove.mutate()}
            >
              <Input
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                className="font-mono"
                placeholder={name}
              />
            </ConfirmDialog>
          </DangerRow>
        )}
      </CardContent>
    </Card>
  )
}

/** One action of the danger zone: title and description left, its button right. */
export function DangerRow({
  title,
  description,
  destructive = false,
  children,
}: {
  title: string
  description: string
  destructive?: boolean
  children: React.ReactNode
}) {
  return (
    <div
      className={`flex items-center justify-between gap-4 rounded-xl border p-4 ${destructive ? "border-destructive/30" : ""}`}
    >
      <div>
        <div className="font-medium">{title}</div>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      {children}
    </div>
  )
}
