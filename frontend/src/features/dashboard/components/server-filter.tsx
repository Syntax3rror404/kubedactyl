import { SearchIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { type ServerFilterState, type StatusFilter } from "@/features/dashboard/lib/server-filter"

/** Search field and status buttons above the server grid. */
export function ServerFilter({
  value,
  onChange,
  counts,
}: {
  value: ServerFilterState
  onChange: (v: ServerFilterState) => void
  counts: Record<StatusFilter, number>
}) {
  const options: { key: StatusFilter; label: string }[] = [
    { key: "all", label: "All" },
    { key: "running", label: "Running" },
    { key: "stopped", label: "Stopped" },
  ]
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <InputGroup className="sm:max-w-xs">
        <InputGroupAddon>
          <SearchIcon />
        </InputGroupAddon>
        <InputGroupInput
          placeholder="Search name, egg or owner…"
          value={value.query}
          onChange={(e) => onChange({ ...value, query: e.target.value })}
        />
      </InputGroup>
      <div className="flex gap-1 rounded-lg border bg-muted/40 p-1">
        {options.map((o) => (
          <Button
            key={o.key}
            size="sm"
            variant={value.status === o.key ? "secondary" : "ghost"}
            className="h-7"
            onClick={() => onChange({ ...value, status: o.key })}
          >
            {o.label}
            <span className="font-mono text-xs text-muted-foreground tabular-nums">{counts[o.key]}</span>
          </Button>
        ))}
      </div>
    </div>
  )
}
