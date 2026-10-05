import { FieldDescription } from "@/components/ui/field"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { userName } from "@/lib/format"
import { useUsers } from "@/lib/queries"

/** Dropdown of the enabled users who can own a server, with the namespace of the chosen one. */
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
  return (
    <>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {users.data
            ?.filter((u) => !u.disabled && u.username !== exclude)
            .map((u) => (
              <SelectItem key={u.username} value={u.username}>
                {userName(u)} <span className="font-mono text-xs text-muted-foreground">{u.username}</span>
              </SelectItem>
            ))}
        </SelectContent>
      </Select>
      {namespace && <FieldDescription className="font-mono text-xs">namespace {namespace}</FieldDescription>}
    </>
  )
}
