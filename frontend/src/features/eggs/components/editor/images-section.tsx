import { ArrowDownIcon, ArrowUpIcon, PlusIcon, Trash2Icon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { SuggestionChip } from "@/features/eggs/components/editor/list-input"
import { move, type FieldErrors, type SpecChange } from "@/features/eggs/lib/egg-draft"
import type { EggSpec } from "@/lib/types"

const stopPresets = [
  { value: "^C", label: "^C (SIGINT)" },
  { value: "^SIGTERM", label: "SIGTERM" },
  { value: "^^C", label: "^^C (SIGKILL)" },
]

/** Placeholders every server gets (besides the egg variables). */
const builtin = ["SERVER_MEMORY", "SERVER_PORT", "SERVER_IP"]

/** Docker images (first = default), startup and stop command. */
export function ImagesSection({
  spec,
  onChange,
  errors,
}: {
  spec: EggSpec
  onChange: SpecChange
  errors: FieldErrors
}) {
  const images = spec.dockerImages
  const setImage = (i: number, patch: Partial<EggSpec["dockerImages"][number]>) =>
    onChange({ dockerImages: images.map((img, j) => (j === i ? { ...img, ...patch } : img)) })
  const placeholders = [...builtin, ...(spec.variables ?? []).map((v) => v.envVariable).filter(Boolean)]
  return (
    <FieldGroup>
      <Field data-invalid={!!errors.dockerImages}>
        <FieldLabel>Docker images</FieldLabel>
        <FieldDescription>Users pick one of them per server; the first is the default.</FieldDescription>
        <div className="space-y-2">
          {images.map((img, i) => (
            <div key={i} className="grid gap-2 sm:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]">
              <Input
                aria-label="Display name"
                value={img.name}
                onChange={(e) => setImage(i, { name: e.target.value })}
                placeholder={i === 0 ? "Java 21 (default)" : "Display name"}
                aria-invalid={!!errors[`dockerImages.${i}.name`]}
              />
              <Input
                aria-label="Image"
                className="font-mono"
                value={img.image}
                onChange={(e) => setImage(i, { image: e.target.value })}
                placeholder="ghcr.io/pelican-eggs/yolks:java_21"
                aria-invalid={!!errors[`dockerImages.${i}.image`]}
              />
              <div className="flex gap-1">
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  title="Move up"
                  disabled={i === 0}
                  onClick={() => onChange({ dockerImages: move(images, i, -1) })}
                >
                  <ArrowUpIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  title="Move down"
                  disabled={i === images.length - 1}
                  onClick={() => onChange({ dockerImages: move(images, i, 1) })}
                >
                  <ArrowDownIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  title="Remove"
                  className="text-destructive"
                  disabled={images.length === 1}
                  onClick={() => onChange({ dockerImages: images.filter((_, j) => j !== i) })}
                >
                  <Trash2Icon />
                </Button>
              </div>
              {(errors[`dockerImages.${i}.name`] || errors[`dockerImages.${i}.image`]) && (
                <FieldError className="sm:col-span-3">
                  {errors[`dockerImages.${i}.image`] ?? `Name ${errors[`dockerImages.${i}.name`]}`}
                </FieldError>
              )}
            </div>
          ))}
        </div>
        {errors.dockerImages && <FieldError>{errors.dockerImages}</FieldError>}
        <div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onChange({ dockerImages: [...images, { name: "", image: "" }] })}
          >
            <PlusIcon />
            Add image
          </Button>
        </div>
      </Field>

      <Field data-invalid={!!errors.startup}>
        <FieldLabel htmlFor="egg-startup">Startup command</FieldLabel>
        <Textarea
          id="egg-startup"
          rows={3}
          spellCheck={false}
          className="font-mono text-xs"
          value={spec.startup}
          onChange={(e) => onChange({ startup: e.target.value })}
          placeholder="java -Xms128M -XX:MaxRAMPercentage=95.0 -jar {{SERVER_JARFILE}}"
        />
        {errors.startup ? (
          <FieldError>{errors.startup}</FieldError>
        ) : (
          <FieldDescription>
            Placeholders in {"{{…}}"} are replaced with the server's variables. Servers can override the command.
          </FieldDescription>
        )}
        <div className="flex flex-wrap gap-1">
          {placeholders.map((p) => (
            <SuggestionChip
              key={p}
              onClick={() =>
                onChange({
                  startup: `${spec.startup}${spec.startup && !spec.startup.endsWith(" ") ? " " : ""}{{${p}}}`,
                })
              }
            >
              {`{{${p}}}`}
            </SuggestionChip>
          ))}
        </div>
      </Field>

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
