import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { SuggestionChip } from "@/features/eggs/components/editor/list-input"
import { envFromName, type FieldErrors } from "@/features/eggs/lib/egg-draft"
import type { EggVariable } from "@/lib/types"

/** Rule suggestions (as in Pelican); arguments after ":" are completed by hand. */
const ruleSuggestions = [
  "required",
  "nullable",
  "string",
  "integer",
  "numeric",
  "boolean",
  "alpha_dash",
  "alpha_num",
  "url",
  "max:20",
  "min:1",
  "between:1024,65535",
  "in:true,false",
  "regex:/^…$/",
]

/** The fields of one egg variable (opened in the variables section). */
export function VariableEditor({
  index,
  variable: v,
  errors,
  onChange,
}: {
  index: number
  variable: EggVariable
  errors: FieldErrors
  onChange: (patch: Partial<EggVariable>) => void
}) {
  const p = `variables.${index}.`
  return (
    <div className="grid gap-4 border-t p-4 sm:grid-cols-2">
      <Field data-invalid={!!errors[p + "name"]}>
        <FieldLabel htmlFor={`var-${index}-name`}>Name</FieldLabel>
        <Input
          id={`var-${index}-name`}
          value={v.name}
          onChange={(e) => {
            // The environment variable follows the name until it is edited by hand.
            const follow = !v.envVariable || v.envVariable === envFromName(v.name)
            onChange({ name: e.target.value, ...(follow ? { envVariable: envFromName(e.target.value) } : {}) })
          }}
          placeholder="Server Jar File"
        />
        {errors[p + "name"] && <FieldError>{errors[p + "name"]}</FieldError>}
      </Field>
      <Field data-invalid={!!errors[p + "envVariable"]}>
        <FieldLabel htmlFor={`var-${index}-env`}>Environment variable</FieldLabel>
        <Input
          id={`var-${index}-env`}
          className="font-mono"
          value={v.envVariable}
          onChange={(e) => onChange({ envVariable: e.target.value })}
          placeholder="SERVER_JARFILE"
        />
        {errors[p + "envVariable"] ? (
          <FieldError>{errors[p + "envVariable"]}</FieldError>
        ) : (
          <FieldDescription>Used as {`{{${v.envVariable || "NAME"}}}`} in the startup command.</FieldDescription>
        )}
      </Field>
      <Field className="sm:col-span-2">
        <FieldLabel htmlFor={`var-${index}-desc`}>Description</FieldLabel>
        <Textarea
          id={`var-${index}-desc`}
          rows={2}
          value={v.description ?? ""}
          onChange={(e) => onChange({ description: e.target.value })}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor={`var-${index}-default`}>Default value</FieldLabel>
        <Input
          id={`var-${index}-default`}
          className="font-mono"
          value={v.defaultValue ?? ""}
          onChange={(e) => onChange({ defaultValue: e.target.value })}
        />
      </Field>
      <Field>
        <FieldLabel>Users can</FieldLabel>
        <div className="flex h-9 items-center gap-5 text-sm">
          <label className="flex items-center gap-2">
            <Checkbox
              checked={!!v.userViewable}
              onCheckedChange={(c) => onChange({ userViewable: !!c, userEditable: c ? v.userEditable : false })}
            />
            view
          </label>
          <label className="flex items-center gap-2">
            <Checkbox
              checked={!!v.userEditable}
              onCheckedChange={(c) => onChange({ userEditable: !!c, userViewable: c ? true : v.userViewable })}
            />
            edit
          </label>
        </div>
      </Field>
      <Field className="sm:col-span-2" data-invalid={!!errors[p + "rules"]}>
        <FieldLabel htmlFor={`var-${index}-rules`}>Rules</FieldLabel>
        <Input
          id={`var-${index}-rules`}
          className="font-mono"
          value={v.rules ?? ""}
          onChange={(e) => onChange({ rules: e.target.value })}
          placeholder="required|string|max:20"
        />
        {errors[p + "rules"] ? (
          <FieldError>{errors[p + "rules"]}</FieldError>
        ) : (
          <FieldDescription>
            Laravel validation rules separated by |, checked when users change the value.
          </FieldDescription>
        )}
        <div className="flex flex-wrap gap-1">
          {ruleSuggestions.map((r) => (
            <SuggestionChip key={r} onClick={() => onChange({ rules: v.rules ? `${v.rules}|${r}` : r })}>
              + {r}
            </SuggestionChip>
          ))}
        </div>
      </Field>
    </div>
  )
}
