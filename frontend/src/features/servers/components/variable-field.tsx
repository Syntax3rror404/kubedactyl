import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import type { EggVariable } from "@/lib/types"

/** Splits Laravel rules like the backend does (regex patterns may contain "|"). */
function rules(v: EggVariable): string[] {
  const out: string[] = []
  const parts = (v.rules ?? "").split("|")
  for (let i = 0; i < parts.length; i++) {
    let p = parts[i]
    if (p.startsWith("regex:")) {
      while (i + 1 < parts.length && !/^regex:(.).*\1[a-z]*$/.test(p)) p += "|" + parts[++i]
    }
    if (p.trim()) out.push(p.trim())
  }
  return out
}

/** Input for one egg variable (text, boolean or select, derived from its rules) with inline validation. */
export function VariableField({
  variable,
  value,
  onChange,
  error,
  disabled,
}: {
  variable: EggVariable
  value: string
  onChange: (v: string) => void
  error?: string
  disabled?: boolean
}) {
  const r = rules(variable)
  const inRule = r.find((x) => x.startsWith("in:"))
  const isBool = r.includes("boolean") || r.includes("bool")
  const required = r.includes("required")
  const id = `var-${variable.envVariable}`

  let control: React.ReactNode
  if (inRule) {
    const options = inRule.slice(3).split(",")
    control = (
      <Select value={value} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder="Select…" />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o} value={o}>
              {o}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )
  } else if (isBool) {
    const on = value === "1" || value.toLowerCase() === "true"
    const trueValue =
      /^(true|false)$/i.test(value) || /^(true|false)$/i.test(variable.defaultValue ?? "") ? "true" : "1"
    const falseValue = trueValue === "true" ? "false" : "0"
    control = (
      <div className="flex h-9 items-center gap-3">
        <Switch
          id={id}
          checked={on}
          disabled={disabled}
          onCheckedChange={(c) => onChange(c ? trueValue : falseValue)}
        />
        <span className="font-mono text-xs text-muted-foreground">{value || "-"}</span>
      </div>
    )
  } else {
    control = (
      <Input
        id={id}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        placeholder={variable.defaultValue || (required ? "required" : "optional")}
        className="font-mono text-sm"
        aria-invalid={!!error}
      />
    )
  }

  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id} className="flex items-center justify-between gap-2">
        <span>
          {variable.name}
          {required && <span className="text-destructive"> *</span>}
        </span>
        <code className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-normal text-muted-foreground">
          {variable.envVariable}
        </code>
      </FieldLabel>
      {control}
      {variable.description && (
        <FieldDescription className="line-clamp-3 whitespace-pre-line">{variable.description}</FieldDescription>
      )}
      {error && <FieldError>{error}</FieldError>}
    </Field>
  )
}
