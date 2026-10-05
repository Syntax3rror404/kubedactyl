import { CopyIcon } from "lucide-react"

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useEggs } from "@/lib/queries"
import type { Egg } from "@/lib/types"

/** Copies settings from another egg once (Pelican/Pterodactyl "Copy Settings From", without inheritance). */
export function CopyFromEgg({
  exclude,
  label,
  onSelect,
}: {
  exclude?: string
  label: string
  onSelect: (egg: Egg) => void
}) {
  const eggs = useEggs().data?.filter((e) => e.metadata.name !== exclude) ?? []
  if (eggs.length === 0) return null
  return (
    <Select value="" onValueChange={(name) => onSelect(eggs.find((e) => e.metadata.name === name)!)}>
      <SelectTrigger size="sm" className="w-fit gap-2" aria-label={label}>
        <CopyIcon className="size-3.5" />
        <SelectValue placeholder={label} />
      </SelectTrigger>
      <SelectContent>
        {eggs.map((e) => (
          <SelectItem key={e.metadata.name} value={e.metadata.name}>
            {e.spec.displayName}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
