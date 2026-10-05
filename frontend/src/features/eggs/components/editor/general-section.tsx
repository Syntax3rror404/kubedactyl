import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { ListInput } from "@/features/eggs/components/editor/list-input"
import type { FieldErrors, SpecChange } from "@/features/eggs/lib/egg-draft"
import { supportedFeatures } from "@/features/servers/lib/egg-features"
import type { EggSpec } from "@/lib/types"

/** Name, author, description, features, tags, hidden files and update URL. */
export function GeneralSection({
  spec,
  onChange,
  errors,
}: {
  spec: EggSpec
  onChange: SpecChange
  errors: FieldErrors
}) {
  return (
    <FieldGroup>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field data-invalid={!!errors.displayName}>
          <FieldLabel htmlFor="egg-name">Name</FieldLabel>
          <Input
            id="egg-name"
            value={spec.displayName}
            maxLength={191}
            onChange={(e) => onChange({ displayName: e.target.value })}
            placeholder="Paper"
          />
          {errors.displayName && <FieldError>{errors.displayName}</FieldError>}
        </Field>
        <Field data-invalid={!!errors.author}>
          <FieldLabel htmlFor="egg-author">Author</FieldLabel>
          <Input
            id="egg-author"
            type="email"
            value={spec.author ?? ""}
            onChange={(e) => onChange({ author: e.target.value })}
            placeholder="you@example.com"
          />
          {errors.author ? (
            <FieldError>{errors.author}</FieldError>
          ) : (
            <FieldDescription>An e-mail address. Pterodactyl and Pelican require it when importing.</FieldDescription>
          )}
        </Field>
      </div>
      <Field>
        <FieldLabel htmlFor="egg-description">Description</FieldLabel>
        <Textarea
          id="egg-description"
          rows={3}
          value={spec.description ?? ""}
          onChange={(e) => onChange({ description: e.target.value })}
        />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field>
          <FieldLabel htmlFor="egg-features">Features</FieldLabel>
          <ListInput
            id="egg-features"
            values={spec.features ?? []}
            onChange={(features) => onChange({ features })}
            suggestions={supportedFeatures}
            mono
            placeholder="eula"
          />
          <FieldDescription>Console helpers that react to known output, like the EULA prompt.</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="egg-tags">Tags</FieldLabel>
          <ListInput
            id="egg-tags"
            values={spec.tags ?? []}
            onChange={(tags) => onChange({ tags })}
            placeholder="minecraft"
          />
          <FieldDescription>Grouping, kept in Pelican exports.</FieldDescription>
        </Field>
      </div>
      <Field>
        <FieldLabel htmlFor="egg-denylist">Hidden files</FieldLabel>
        <ListInput
          id="egg-denylist"
          values={spec.fileDenylist ?? []}
          onChange={(fileDenylist) => onChange({ fileDenylist })}
          mono
          placeholder="*.key"
        />
        <FieldDescription>
          File names or patterns the file manager hides from users (egg file_denylist).
        </FieldDescription>
      </Field>
      <Field data-invalid={!!errors["source"]}>
        <FieldLabel htmlFor="egg-update-url">Update URL</FieldLabel>
        <Input
          id="egg-update-url"
          type="url"
          className="font-mono"
          value={spec.source?.updateUrl ?? ""}
          onChange={(e) =>
            onChange({
              source: {
                ...spec.source,
                updateUrl: e.target.value,
                autoUpdate: !!e.target.value && spec.source?.autoUpdate,
              },
            })
          }
          placeholder="https://raw.githubusercontent.com/…/egg.yaml"
        />
        <FieldDescription>Source of “Update from URL”. Empty: the egg is not updated.</FieldDescription>
      </Field>
      <Field>
        <div className="flex items-center gap-2">
          <Switch
            id="egg-auto-update"
            checked={!!spec.source?.autoUpdate && !!spec.source.updateUrl}
            disabled={!spec.source?.updateUrl}
            onCheckedChange={(autoUpdate) => onChange({ source: { ...spec.source, autoUpdate } })}
          />
          <FieldLabel htmlFor="egg-auto-update">Update automatically</FieldLabel>
        </div>
        <FieldDescription>Checked every hour. A newer file replaces changes made in the panel.</FieldDescription>
      </Field>
    </FieldGroup>
  )
}
