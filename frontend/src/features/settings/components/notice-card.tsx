import { MegaphoneIcon } from "lucide-react"

import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Textarea } from "@/components/ui/textarea"
import { SettingsCard } from "@/features/settings/components/settings-card"

const MAX = 2000

/** Notice that users see every time they open one of their servers. */
export function NoticeCard({
  notice,
  onChange,
  error,
}: {
  notice: string
  onChange: (notice: string) => void
  error?: string
}) {
  const tooLong = notice.length > MAX
  return (
    <SettingsCard
      icon={MegaphoneIcon}
      title="Server notice"
      description="Shown as a dialog whenever users open one of their servers. Empty shows nothing."
    >
      <Field data-invalid={tooLong || !!error}>
        <FieldLabel htmlFor="notice">Notice</FieldLabel>
        <Textarea
          id="notice"
          rows={4}
          value={notice}
          onChange={(e) => onChange(e.target.value)}
          placeholder="e.g. Maintenance on Sunday 22:00, servers restart automatically."
          aria-invalid={tooLong || !!error}
        />
        {error ? (
          <FieldError>{error}</FieldError>
        ) : (
          <FieldDescription className={tooLong ? "text-destructive" : undefined}>
            Plain text, line breaks are kept · {notice.length} / {MAX}
          </FieldDescription>
        )}
      </Field>
    </SettingsCard>
  )
}
