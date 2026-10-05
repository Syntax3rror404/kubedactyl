import { toast } from "sonner"

import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"
import { ConfigFilesEditor } from "@/features/eggs/components/editor/config-files-editor"
import { CopyFromEgg } from "@/features/eggs/components/editor/copy-from-egg"
import { ListInput } from "@/features/eggs/components/editor/list-input"
import type { FieldErrors, SpecChange } from "@/features/eggs/lib/egg-draft"
import type { EggSpec } from "@/lib/types"

/** Startup detection, ANSI stripping and config files. */
export function ProcessSection({
  spec,
  onChange,
  errors,
  self,
}: {
  spec: EggSpec
  onChange: SpecChange
  errors: FieldErrors
  self?: string
}) {
  const doneErrors = Object.entries(errors).filter(([k]) => k.startsWith("startupDone."))
  return (
    <FieldGroup>
      <div className="flex justify-end">
        <CopyFromEgg
          exclude={self}
          label="Copy settings from…"
          onSelect={(egg) => {
            onChange({
              stop: egg.spec.stop ?? "",
              startupDone: [...(egg.spec.startupDone ?? [])],
              stripAnsi: !!egg.spec.stripAnsi,
              configFiles: structuredClone(egg.spec.configFiles ?? []),
            })
            toast.success(`Copied stop command, startup detection and config files from ${egg.spec.displayName}`)
          }}
        />
      </div>
      <Field data-invalid={doneErrors.length > 0}>
        <FieldLabel htmlFor="egg-done">Running when the console shows</FieldLabel>
        <ListInput
          id="egg-done"
          values={spec.startupDone ?? []}
          onChange={(startupDone) => onChange({ startupDone })}
          separator=""
          mono
          placeholder=")! For help, type "
        />
        {doneErrors.length ? (
          <FieldError>{doneErrors[0][1]}</FieldError>
        ) : (
          <FieldDescription>
            Output that marks the server as running (“regex:” for a regular expression). Empty: running once the
            container runs.
          </FieldDescription>
        )}
      </Field>
      <Field orientation="horizontal">
        <Switch
          id="egg-strip-ansi"
          checked={!!spec.stripAnsi}
          onCheckedChange={(stripAnsi) => onChange({ stripAnsi })}
        />
        <div>
          <FieldLabel htmlFor="egg-strip-ansi">Strip colors before matching</FieldLabel>
          <FieldDescription>
            Removes ANSI color codes from the output before looking for the texts above.
          </FieldDescription>
        </div>
      </Field>
      <Field>
        <FieldLabel>Config files</FieldLabel>
        <FieldDescription>
          Values written into files before every start, e.g. the port. Placeholders: {"{{server.build.default.port}}"},{" "}
          {"{{server.build.memory}}"}, {"{{server.build.env.VARIABLE}}"}, {"{{env.VARIABLE}}"}.
        </FieldDescription>
        <ConfigFilesEditor
          files={spec.configFiles ?? []}
          onChange={(configFiles) => onChange({ configFiles })}
          errors={errors}
        />
      </Field>
    </FieldGroup>
  )
}
