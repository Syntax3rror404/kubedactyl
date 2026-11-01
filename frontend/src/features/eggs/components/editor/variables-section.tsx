import { useState } from "react"
import { ChevronRightIcon, PlusIcon, VariableIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { ReorderButtons } from "@/features/eggs/components/editor/named-list"
import { VariableEditor } from "@/features/eggs/components/editor/variable-editor"
import { move, patchAt, removeAt, type FieldErrors, type SpecChange } from "@/features/eggs/lib/egg-draft"
import type { EggSpec, EggVariable } from "@/lib/types"
import { cn } from "@/lib/utils"

/**
 * Egg variables: order, names, defaults, permissions and validation rules. One opens at a time; those with errors
 * are open until closed.
 */
export function VariablesSection({
  spec,
  onChange,
  errors,
}: {
  spec: EggSpec
  onChange: SpecChange
  errors: FieldErrors
}) {
  const vars = spec.variables ?? []
  // The variable the user opened or closed last; the others are open while they have errors.
  const [toggled, setToggled] = useState<{ index: number; open: boolean } | null>(null)
  const set = (i: number, patch: Partial<EggVariable>) => onChange({ variables: patchAt(vars, i, patch) })
  const reorder = (i: number, by: -1 | 1) => {
    onChange({ variables: move(vars, i, by) }, true)
    if (toggled?.index === i) setToggled({ ...toggled, index: i + by })
    else if (toggled?.index === i + by) setToggled({ ...toggled, index: i })
  }
  return (
    <div className="space-y-2">
      {vars.length === 0 && (
        <Empty className="rounded-xl border border-dashed py-10">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <VariableIcon />
            </EmptyMedia>
            <EmptyTitle>No variables</EmptyTitle>
            <EmptyDescription>
              Variables become environment variables and {"{{placeholders}}"} in the startup command, e.g. the server
              version.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}
      {vars.map((v, i) => {
        const p = `variables.${i}.`
        const invalid = Object.keys(errors).some((k) => k.startsWith(p))
        const expanded = toggled?.index === i ? toggled.open : invalid
        return (
          <div key={i} className={cn("rounded-xl border", invalid && "border-destructive/60")}>
            <div className="flex items-center gap-2 p-2">
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-1 py-1 text-left"
                onClick={() => setToggled({ index: i, open: !expanded })}
                aria-expanded={expanded}
              >
                <ChevronRightIcon
                  className={cn("size-4 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-90")}
                />
                <span className="truncate font-medium">{v.name || "New variable"}</span>
                {v.envVariable && (
                  <code className="truncate text-xs text-muted-foreground">{`{{${v.envVariable}}}`}</code>
                )}
                <span className="ml-auto hidden gap-1 sm:flex">
                  {!v.userViewable && <Badge variant="outline">hidden</Badge>}
                  {v.userViewable && !v.userEditable && <Badge variant="outline">read-only</Badge>}
                </span>
              </button>
              <ReorderButtons
                index={i}
                count={vars.length}
                size="icon-sm"
                onMove={(by) => reorder(i, by)}
                onRemove={() => {
                  onChange({ variables: removeAt(vars, i) }, true)
                  setToggled(null)
                }}
              />
            </div>
            {expanded && <VariableEditor index={i} variable={v} errors={errors} onChange={(patch) => set(i, patch)} />}
          </div>
        )
      })}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => {
          onChange({
            variables: [
              ...vars,
              {
                name: "",
                envVariable: "",
                defaultValue: "",
                userViewable: true,
                userEditable: true,
                rules: "required|string|max:20",
                fieldType: "text",
              },
            ],
          })
          setToggled({ index: vars.length, open: true })
        }}
      >
        <PlusIcon />
        Add variable
      </Button>
    </div>
  )
}
