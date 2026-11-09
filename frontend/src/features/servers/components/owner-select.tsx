import { FieldDescription } from "@/components/ui/field"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { userName } from "@/lib/format"
import { useUsers } from "@/lib/queries"

/** Dropdown of the enabled users who can own a server (no subusers), with the namespace of the chosen one. */
export function OwnerSelect({
  id,
  value,
  onChange,
  exclude,
  placeholder,
}: {
  id: string
  value: string
  onChange: (username: string) => void
  /** A user not to offer (e.g. the current owner). */
  exclude?: string
  placeholder?: string
}) {
  const users = useUsers()
  const namespace = users.data?.find((u) => u.username === value)?.namespace
  const owners = (users.data ?? [])
    .filter((u) => !u.disabled && !u.owner && u.username !== exclude)
    .map((u) => ({
      value: u.username,
      label: (
        <>
          {userName(u)} <span className="font-mono text-xs text-muted-foreground">{u.username}</span>
        </>
      ),
    }))
  return (
    <>
      <Select items={owners} value={value} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {owners.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {namespace && <FieldDescription className="font-mono text-xs">namespace {namespace}</FieldDescription>}
    </>
  )
}
