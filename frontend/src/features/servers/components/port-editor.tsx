import { useState } from "react"
import { PlusIcon, XIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

/**
 * Edits the port list; the first port is the primary one (SERVER_PORT). At least one port is
 * required: the parent blocks saving while the list is empty. A typed port is also added when
 * the field loses focus, so it cannot be forgotten.
 */
export function PortEditor({
  ports,
  onChange,
  id,
}: {
  ports: number[]
  onChange: (ports: number[]) => void
  id?: string
}) {
  const [draft, setDraft] = useState("")
  const add = () => {
    const p = Number(draft)
    if (Number.isInteger(p) && p > 0 && p < 65536 && !ports.includes(p)) onChange([...ports, p])
    setDraft("")
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      {ports.map((p, i) => (
        <Badge key={p} variant={i === 0 ? "default" : "secondary"} className="gap-1 py-1 pr-1 font-mono">
          {p}
          {i === 0 && <span className="font-sans text-[10px] opacity-70">primary</span>}
          <button
            type="button"
            className="rounded-full p-0.5 hover:bg-black/10"
            onClick={() => onChange(ports.filter((x) => x !== p))}
            aria-label={`Remove port ${p}`}
          >
            <XIcon className="size-3" />
          </button>
        </Badge>
      ))}
      <div className="flex items-center gap-1">
        <Input
          id={id}
          value={draft}
          inputMode="numeric"
          onChange={(e) => setDraft(e.target.value.replace(/\D/g, ""))}
          onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), add())}
          onBlur={() => draft && add()}
          placeholder={ports.length ? "Add port" : "e.g. 25565"}
          aria-invalid={ports.length === 0}
          className="h-7 w-28 font-mono text-xs"
        />
        <Button type="button" size="icon-sm" variant="outline" onClick={add} disabled={!draft} aria-label="Add port">
          <PlusIcon />
        </Button>
      </div>
    </div>
  )
}
