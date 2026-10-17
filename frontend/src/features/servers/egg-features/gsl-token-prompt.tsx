import { useState } from "react"
import { toast } from "sonner"

import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { FeaturePrompt } from "@/features/servers/egg-features/feature-prompt"
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
  const variable = egg.spec.variables?.find((v) => v.envVariable === VARIABLE)
  const editable = !!variable && (isAdmin || !!variable.userEditable)
  const [token, setToken] = useState("")
  const update = useUpdateAndStartServer(server.metadata.name, {
    onSuccess: () => {
      dismiss()
      setToken("")
      toast.success("GSL token updated", { description: "The server restarts." })
    },
    onError: failed("update the GSL token"),
  })
  const save = () =>
    update.mutate({
      changes: { environment: { ...server.spec.environment, [VARIABLE]: token.trim() } },
      signal: active ? "restart" : "start",
    })
  return (
    <FeaturePrompt
      open={!!line && !running}
      onClose={dismiss}
      title="Invalid GSL token"
      description={
        <>
          Your GSL token seems invalid or expired.{" "}
          {editable ? "Enter a new one below, or leave it empty to remove it." : "Ask an administrator to update it."}
        </>
      }
      confirm={editable ? { label: "Update GSL token", pending: update.isPending, onConfirm: save } : undefined}
    >
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
    </FeaturePrompt>
  )
}
