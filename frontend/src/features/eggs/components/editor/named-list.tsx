import type { ReactNode } from "react"
import { ArrowDownIcon, ArrowUpIcon, PlusIcon, Trash2Icon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { move, type FieldErrors } from "@/features/eggs/lib/egg-draft"

type Entry<K extends string> = { name: string } & Record<K, string>

/**
 * A list of named values users pick from by name (docker images, startup commands): the first is the
 * default, entries can be moved and removed (at least one stays). Errors are keyed "<path>.<i>.<key>".
 */
export function NamedList<K extends string>({
  label,
  description,
  path,
  valueKey,
  items,
  onChange,
  errors,
  namePlaceholder,
  valuePlaceholder,
  addLabel,
  multiline,
  onFocusValue,
  children,
}: {
  label: string
  description: string
  path: string
  valueKey: K
  items: Entry<K>[]
  onChange: (items: Entry<K>[]) => void
  errors: FieldErrors
  namePlaceholder: (index: number) => string
  valuePlaceholder: string
  addLabel: string
  /** A textarea for long values (commands). */
  multiline?: boolean
  onFocusValue?: (index: number) => void
  children?: ReactNode
}) {
  const set = (i: number, patch: Partial<Entry<K>>) =>
    onChange(items.map((item, j) => (j === i ? { ...item, ...patch } : item)))
  const Value = multiline ? Textarea : Input
  return (
    <Field data-invalid={!!errors[path]}>
      <FieldLabel>{label}</FieldLabel>
      <FieldDescription>{description}</FieldDescription>
      <div className="space-y-2">
        {items.map((item, i) => {
          const nameError = errors[`${path}.${i}.name`]
          const valueError = errors[`${path}.${i}.${valueKey}`]
          return (
            <div key={i} className="grid gap-2 sm:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]">
              <Input
                aria-label="Display name"
                value={item.name}
                onChange={(e) => set(i, { name: e.target.value } as Partial<Entry<K>>)}
                placeholder={namePlaceholder(i)}
                aria-invalid={!!nameError}
              />
              <Value
                aria-label={label}
                className={multiline ? "min-h-9 font-mono text-xs" : "font-mono"}
                rows={multiline ? 2 : undefined}
                spellCheck={false}
                value={item[valueKey]}
                onChange={(e) => set(i, { [valueKey]: e.target.value } as Partial<Entry<K>>)}
                onFocus={() => onFocusValue?.(i)}
                placeholder={valuePlaceholder}
                aria-invalid={!!valueError}
              />
              <div className="flex gap-1">
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  title="Move up"
                  disabled={i === 0}
                  onClick={() => onChange(move(items, i, -1))}
                >
                  <ArrowUpIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  title="Move down"
                  disabled={i === items.length - 1}
                  onClick={() => onChange(move(items, i, 1))}
                >
                  <ArrowDownIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  title="Remove"
                  className="text-destructive"
                  disabled={items.length === 1}
                  onClick={() => onChange(items.filter((_, j) => j !== i))}
                >
                  <Trash2Icon />
                </Button>
              </div>
              {(nameError || valueError) && (
                <FieldError className="sm:col-span-3">{valueError ?? `Name ${nameError}`}</FieldError>
              )}
            </div>
          )
        })}
      </div>
      {errors[path] && <FieldError>{errors[path]}</FieldError>}
      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onChange([...items, { name: "", [valueKey]: "" } as Entry<K>])}
        >
          <PlusIcon />
          {addLabel}
        </Button>
      </div>
      {children}
    </Field>
  )
}
