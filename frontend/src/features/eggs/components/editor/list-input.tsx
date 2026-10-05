import { useState } from "react"
import { XIcon } from "lucide-react"

import { cn } from "@/lib/utils"

/** Values as removable chips; Enter or comma adds the typed value. */
export function ListInput({
  id,
  values,
  onChange,
  placeholder,
  suggestions = [],
  mono,
  separator = ",",
}: {
  id?: string
  values: string[]
  onChange: (values: string[]) => void
  placeholder?: string
  suggestions?: readonly string[]
  mono?: boolean
  /** Key that also adds the value (besides Enter); "" for none, e.g. when values contain commas. */
  separator?: string
}) {
  const [text, setText] = useState("")
  const add = (value: string) => {
    const v = value.trim()
    if (v && !values.includes(v)) onChange([...values, v])
    setText("")
  }
  const open = suggestions.filter((s) => !values.includes(s))
  return (
    <div className="space-y-2">
      <div className="flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border bg-transparent px-2 py-1.5 shadow-xs focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50 dark:bg-input/30">
        {values.map((v) => (
          <span
            key={v}
            className={cn(
              "inline-flex items-center gap-1 rounded-md bg-secondary px-2 py-0.5 text-xs",
              mono && "font-mono",
            )}
          >
            {v}
            <button
              type="button"
              className="text-muted-foreground hover:text-foreground"
              aria-label={`Remove ${v}`}
              onClick={() => onChange(values.filter((x) => x !== v))}
            >
              <XIcon className="size-3" />
            </button>
          </span>
        ))}
        <input
          id={id}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || (separator && e.key === separator)) {
              e.preventDefault()
              add(text)
            } else if (e.key === "Backspace" && !text && values.length) {
              onChange(values.slice(0, -1))
            }
          }}
          onBlur={() => add(text)}
          placeholder={values.length ? "" : placeholder}
          className={cn(
            "min-w-32 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground",
            mono && "font-mono",
          )}
        />
      </div>
      {open.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {open.map((s) => (
            <SuggestionChip key={s} onClick={() => add(s)}>
              + {s}
            </SuggestionChip>
          ))}
        </div>
      )}
    </div>
  )
}

/** A dashed chip that inserts a suggested value. */
export function SuggestionChip({ onClick, children }: { onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-md border border-dashed px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground hover:border-solid hover:text-foreground"
    >
      {children}
    </button>
  )
}
