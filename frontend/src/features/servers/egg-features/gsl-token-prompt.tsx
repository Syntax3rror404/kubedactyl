import { useState } from "react"
import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useConsoleMatch, type Subscribe } from "@/features/servers/egg-features/use-console-match"
import { failed } from "@/lib/notify"
import { useUpdateAndStartServer } from "@/lib/queries"
import type { Egg, GameServer } from "@/lib/types"

const patterns = ["(gsl token expired)", "(account not found)"]
const VARIABLE = "STEAM_ACC"

/** Egg feature "gsl_token": the Steam game server login token is invalid, enter a new one. */
export function GslTokenPrompt({
  server,
  egg,
  subscribe,
  active,
  running,
  isAdmin,
}: {
  server: GameServer
  egg: Egg
  subscribe: Subscribe
  active: boolean
  running: boolean
  isAdmin: boolean
}) {
  const { line, dismiss } = useConsoleMatch(subscribe, patterns)
  const name = server.metadata.name
  const variable = egg.spec.variables?.find((v) => v.envVariable === VARIABLE)
  const editable = !!variable && (isAdmin || !!variable.userEditable)
  const [token, setToken] = useState("")
  const update = useUpdateAndStartServer(name, {
    onSuccess: () => {
      dismiss()
      setToken("")
      toast.success("GSL token updated", { description: "The server restarts." })
    },
    onError: failed("update the GSL token"),
  })
  return (
    <AlertDialog open={!!line && !running} onOpenChange={(o) => !o && dismiss()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Invalid GSL token</AlertDialogTitle>
          <AlertDialogDescription>
            Your GSL token seems invalid or expired.{" "}
            {editable ? "Enter a new one below, or leave it empty to remove it." : "Ask an administrator to update it."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {editable && (
          <Field>
            <FieldLabel htmlFor="gsl-token">GSL token</FieldLabel>
            <Input
              id="gsl-token"
              className="font-mono"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              autoComplete="off"
            />
            <FieldDescription>
              Create one at{" "}
              <a
                className="underline underline-offset-4"
                href="https://steamcommunity.com/dev/managegameservers"
                target="_blank"
                rel="noreferrer"
              >
                steamcommunity.com/dev/managegameservers
              </a>
              .
            </FieldDescription>
          </Field>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          {editable && (
            <AlertDialogAction
              disabled={update.isPending}
              onClick={(e) => {
                e.preventDefault()
                update.mutate({
                  changes: { environment: { ...server.spec.environment, [VARIABLE]: token.trim() } },
                  signal: active ? "restart" : "start",
                })
              }}
            >
              Update GSL token
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
