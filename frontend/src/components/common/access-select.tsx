import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { ShareAccess } from "@/lib/types"

/** What a subuser may do on a server; "none" is no share. */
export type AccessChoice = ShareAccess | "none"

const choices: { value: AccessChoice; label: string }[] = [
  { value: "none", label: "No access" },
  { value: "view", label: "View" },
  { value: "control", label: "Control" },
]

/** Picks the access of a subuser to a server: none, view (console output) or control (also start, stop, commands). */
export function AccessSelect({
  value,
  onChange,
  label,
}: {
  value: AccessChoice
  onChange: (value: AccessChoice) => void
  label: string
}) {
  return (
    <Select value={value} onValueChange={(v) => onChange(v as AccessChoice)}>
      <SelectTrigger className="w-32" aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {choices.map((c) => (
          <SelectItem key={c.value} value={c.value}>
            {c.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
