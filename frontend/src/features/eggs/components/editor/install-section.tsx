import { toast } from "sonner"

import { CodeEditor } from "@/components/common/code-editor"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { CopyFromEgg } from "@/features/eggs/components/editor/copy-from-egg"
import type { SpecChange } from "@/features/eggs/lib/egg-draft"
import type { EggSpec } from "@/lib/types"

const entrypoints = ["bash", "ash", "/bin/bash", "sh"]

/** Installer image, entrypoint and the install script. */
export function InstallSection({ spec, onChange, self }: { spec: EggSpec; onChange: SpecChange; self?: string }) {
  const install = spec.install ?? {}
  const set = (patch: Partial<NonNullable<EggSpec["install"]>>) => onChange({ install: { ...install, ...patch } })
  return (
    <FieldGroup>
      <div className="flex justify-end">
        <CopyFromEgg
          exclude={self}
          label="Copy script from…"
          onSelect={(egg) => {
            set({ ...egg.spec.install })
            toast.success(`Copied the install script of ${egg.spec.displayName}`)
          }}
        />
      </div>
      <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,14rem)]">
        <Field>
          <FieldLabel htmlFor="egg-install-container">Script container</FieldLabel>
          <Input
            id="egg-install-container"
            className="font-mono"
            value={install.container ?? ""}
            onChange={(e) => set({ container: e.target.value })}
            placeholder="ghcr.io/pelican-eggs/installers:debian"
          />
          <FieldDescription>Image the script runs in; the server files are mounted at /mnt/server.</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="egg-install-entry">Entrypoint</FieldLabel>
          <Input
            id="egg-install-entry"
            className="font-mono"
            list="egg-entrypoints"
            value={install.entrypoint ?? ""}
            onChange={(e) => set({ entrypoint: e.target.value })}
            placeholder="bash"
          />
          <datalist id="egg-entrypoints">
            {entrypoints.map((e) => (
              <option key={e} value={e} />
            ))}
          </datalist>
          <FieldDescription>Shell that runs the script.</FieldDescription>
        </Field>
      </div>
      <Field>
        <FieldLabel>Script</FieldLabel>
        <CodeEditor value={install.script ?? ""} onChange={(script) => set({ script })} language="sh" height="50vh" />
        <FieldDescription>
          Egg variables are available as environment variables. Leave it empty to install nothing.
        </FieldDescription>
      </Field>
    </FieldGroup>
  )
}
