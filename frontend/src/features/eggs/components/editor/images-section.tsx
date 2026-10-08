import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { SuggestionChip } from "@/features/eggs/components/editor/list-input"
import { NamedList } from "@/features/eggs/components/editor/named-list"
import type { FieldErrors, SpecChange } from "@/features/eggs/lib/egg-draft"
import type { EggSpec } from "@/lib/types"

const stopPresets = [
  { value: "^C", label: "^C (SIGINT)" },
  { value: "^SIGTERM", label: "SIGTERM" },
  { value: "^^C", label: "^^C (SIGKILL)" },
]

/** Placeholders every server gets (besides the egg variables). */
const builtin = ["SERVER_MEMORY", "SERVER_PORT", "SERVER_IP"]

/** Docker images and startup commands (first = default) and the stop command. */
export function ImagesSection({
  spec,
  onChange,
  errors,
}: {
  spec: EggSpec
  onChange: SpecChange
  errors: FieldErrors
}) {
  const commands = spec.startupCommands ?? []
  // The placeholder chips add to the command that had the focus last.
  const [active, setActive] = useState(0)
  const insert = (text: string) =>
    onChange({
      startupCommands: commands.map((c, i) =>
        i === Math.min(active, commands.length - 1)
          ? { ...c, command: `${c.command}${c.command && !c.command.endsWith(" ") ? " " : ""}${text}` }
          : c,
      ),
    })
  const placeholders = [...builtin, ...(spec.variables ?? []).map((v) => v.envVariable).filter(Boolean)]
  return (
    <FieldGroup>
      <NamedList
        label="Docker images"
        description="Users pick one of them per server; the first is the default."
        path="dockerImages"
        valueKey="image"
        items={spec.dockerImages}
        onChange={(dockerImages) => onChange({ dockerImages })}
        errors={errors}
        namePlaceholder={(i) => (i === 0 ? "Java 21 (default)" : "Display name")}
        valuePlaceholder="ghcr.io/pelican-eggs/yolks:java_21"
        addLabel="Add image"
      />

      <NamedList
        label="Startup commands"
        description="Users pick one of them per server; the first is the default. Placeholders in {{…}} are replaced with the server's variables."
        path="startupCommands"
        valueKey="command"
        items={commands}
        onChange={(startupCommands) => onChange({ startupCommands })}
        errors={errors}
        namePlaceholder={(i) => (i === 0 ? "Default" : "Display name")}
        valuePlaceholder="java -Xms128M -XX:MaxRAMPercentage=95.0 -jar {{SERVER_JARFILE}}"
        addLabel="Add startup command"
        multiline
        onFocusValue={setActive}
      >
        <div className="flex flex-wrap gap-1">
          {placeholders.map((p) => (
            <SuggestionChip key={p} onClick={() => insert(`{{${p}}}`)}>
              {`{{${p}}}`}
            </SuggestionChip>
          ))}
        </div>
      </NamedList>

      <Field>
        <FieldLabel htmlFor="egg-stop">Stop command</FieldLabel>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            id="egg-stop"
            className="font-mono sm:max-w-xs"
            value={spec.stop ?? ""}
            onChange={(e) => onChange({ stop: e.target.value })}
            placeholder="stop"
          />
          <div className="flex flex-wrap gap-1">
            {stopPresets.map((p) => (
              <Button
                key={p.value}
                type="button"
                size="sm"
                variant={spec.stop === p.value ? "secondary" : "ghost"}
                onClick={() => onChange({ stop: p.value })}
              >
                {p.label}
              </Button>
            ))}
          </div>
        </div>
        <FieldDescription>
          A console command (e.g. “stop”), or a signal starting with ^. Empty stops the container gracefully.
        </FieldDescription>
      </Field>
    </FieldGroup>
  )
}
