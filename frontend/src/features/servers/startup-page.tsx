import { useMemo } from "react"
import { RotateCcwIcon, SaveIcon } from "lucide-react"
import { useOutletContext } from "react-router"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { StartupCommandField } from "@/features/servers/components/startup-command-field"
import { StartupPreview } from "@/features/servers/components/startup-preview"
import { VariableField } from "@/features/servers/components/variable-field"
import { defaultEnvironment, imageOptions, startupCommandOf } from "@/features/servers/lib/startup"
import type { ServerContext } from "@/features/servers/server-layout"
import { useAuth } from "@/hooks/use-auth"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useUpdateServer } from "@/lib/queries"
import type { UpdateServerRequest } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/** /servers/:server/startup: startup command preview, docker image and the egg variables. */
export function StartupPage() {
  const { server, egg } = useOutletContext<ServerContext>()
  const name = server.metadata.name
  const { isAdmin } = useAuth()
  const spec = server.spec
  const initial = useMemo(
    () => ({
      startup: spec.startup ?? "",
      startupName: spec.startupName ?? "",
      image: spec.image,
      env: spec.environment ?? ({} as Record<string, string>),
    }),
    [spec],
  )
  const { draft, set, setDraft, dirty } = useDraft(initial)
  const { startup, startupName, image, env } = draft

  const knownImage = egg?.spec.dockerImages.some((i) => i.image === image)
  const images = [
    ...imageOptions(egg?.spec.dockerImages ?? []),
    ...(isAdmin || !knownImage ? [{ value: "custom", label: "Custom…" }] : []),
  ]
  // Users may only change editable variables and pick one of the egg images.
  const editable = (key: string) =>
    isAdmin || !!egg?.spec.variables?.some((v) => v.envVariable === key && v.userEditable)
  const onlyEditable = (vars: Record<string, string>) =>
    Object.fromEntries(Object.entries(vars).filter(([k]) => editable(k)))

  const changes = (): UpdateServerRequest =>
    isAdmin ? { startup, startupName, image, environment: env } : { startupName, image, environment: onlyEditable(env) }
  const save = useUpdateServer(name, {
    onSuccess: () => toast.success("Startup settings saved", { description: "Changes apply on the next start." }),
    onError: failed("save the startup settings"),
  })
  const errors = fieldErrors(save.error)

  return (
    <div className="grid gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Startup command</CardTitle>
          <CardDescription>
            {isAdmin
              ? "Pick a command of the egg or enter a custom one. "
              : startup
                ? "Set by your administrator. "
                : "Pick one of the commands of the egg. "}
            {"{{VARIABLES}}"} are replaced inside the container.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <StartupCommandField
            egg={egg?.spec}
            value={{ startup, startupName }}
            onChange={(v) => setDraft((d) => ({ ...d, ...v }))}
            isAdmin={isAdmin}
          />
          <StartupPreview
            startup={startupCommandOf(egg?.spec, draft)}
            env={env}
            memory={server.spec.resources.memoryMiB}
            port={server.spec.ports[0]}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Docker image</CardTitle>
          <CardDescription>
            {isAdmin
              ? "Pick one of the egg images or enter a custom one."
              : knownImage
                ? "Pick one of the egg images."
                : "Set by your administrator."}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <FieldGroup className="grid gap-5 md:grid-cols-2">
            <Field>
              <FieldLabel>Image</FieldLabel>
              <Select
                items={images}
                value={knownImage ? image : "custom"}
                onValueChange={(v) => v && v !== "custom" && set("image", v)}
                disabled={!isAdmin && !knownImage}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {images.map((i) => (
                    <SelectItem key={i.value} value={i.value}>
                      {i.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>Image reference</FieldLabel>
              <Input
                value={image}
                onChange={(e) => set("image", e.target.value)}
                readOnly={!isAdmin}
                className="font-mono text-sm"
              />
              <FieldDescription>Pulled by Kubernetes on the next start.</FieldDescription>
            </Field>
          </FieldGroup>
        </CardContent>
      </Card>

      {egg?.spec.variables?.length ? (
        <Card>
          <CardHeader>
            <CardTitle>Variables</CardTitle>
            <CardDescription>Validated against the rules of the egg.</CardDescription>
          </CardHeader>
          <CardContent>
            <FieldGroup className="grid gap-6 md:grid-cols-2">
              {egg.spec.variables.map((v) => (
                <VariableField
                  key={v.envVariable}
                  variable={v}
                  value={env[v.envVariable] ?? v.defaultValue ?? ""}
                  error={errors[v.envVariable]}
                  disabled={!isAdmin && !v.userEditable}
                  onChange={(val) => set("env", { ...env, [v.envVariable]: val })}
                />
              ))}
            </FieldGroup>
          </CardContent>
          <CardFooter className="justify-between border-t">
            <Button
              variant="ghost"
              onClick={() => set("env", { ...env, ...onlyEditable(defaultEnvironment(egg.spec)) })}
            >
              <RotateCcwIcon />
              Reset to defaults
            </Button>
          </CardFooter>
        </Card>
      ) : null}

      <div className="sticky bottom-4 flex justify-end">
        <Button
          size="lg"
          className="shadow-lg"
          disabled={!dirty || save.isPending}
          onClick={() => save.mutate(changes())}
        >
          {save.isPending ? <Spinner /> : <SaveIcon />}
          Save startup settings
        </Button>
      </div>
    </div>
  )
}
