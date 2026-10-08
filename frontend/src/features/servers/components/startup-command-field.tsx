import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { pickedCommand, type StartupChoice } from "@/features/servers/lib/startup"
import type { EggSpec } from "@/lib/types"

/**
 * The startup command of a server: one of the egg's commands or, for admins, a custom one; the command shows
 * below and can be edited only when custom. A custom command set by an admin is read-only for users.
 */
export function StartupCommandField({
  egg,
  value,
  onChange,
  isAdmin,
}: {
  egg?: EggSpec
  value: StartupChoice
  onChange: (value: StartupChoice) => void
  isAdmin: boolean
}) {
  const picked = pickedCommand(egg, value)
  const custom = value.startup !== ""
  const pick = (name: string) =>
    onChange(
      name === "custom"
        ? { ...value, startup: value.startup || picked?.command || "" }
        : { startup: "", startupName: name },
    )
  return (
    <div className="space-y-3">
      <Select value={custom ? "custom" : picked?.name} onValueChange={pick} disabled={!isAdmin && custom}>
        <SelectTrigger className="w-full" aria-label="Startup command">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {egg?.startupCommands?.map((c) => (
            <SelectItem key={c.name} value={c.name}>
              {c.name}
            </SelectItem>
          ))}
          {(isAdmin || custom) && <SelectItem value="custom">Custom startup</SelectItem>}
        </SelectContent>
      </Select>
      <Textarea
        aria-label="Command"
        value={custom ? value.startup : (picked?.command ?? "")}
        onChange={(e) => onChange({ ...value, startup: e.target.value })}
        readOnly={!isAdmin || !custom}
        className="min-h-20 font-mono text-sm"
      />
    </div>
  )
}
