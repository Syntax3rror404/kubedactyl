import { MegaphoneIcon } from "lucide-react"

import { Textarea } from "@/components/ui/textarea"
import { SettingsCard, SettingsField } from "@/features/settings/components/settings-card"
import { tooLong, type SettingsCardProps } from "@/features/settings/lib/settings-draft"

const MAX_NOTICE = 2000

/** Notice that users see every time they open one of their servers. */
export function NoticeCard({ draft, onChange, errors }: SettingsCardProps) {
  const notice = draft.serverNotice
  return (
    <SettingsCard
      icon={MegaphoneIcon}
      title="Server notice"
      description="Shown as a dialog whenever users open one of their servers. Empty shows nothing."
    >
      <SettingsField
        id="notice"
        label="Notice"
        hint={`Plain text, line breaks are kept · ${notice.length} / ${MAX_NOTICE}`}
        error={errors.serverNotice ?? tooLong(notice, MAX_NOTICE)}
      >
        {(control) => (
          <Textarea
            {...control}
            rows={4}
            value={notice}
            onChange={(e) => onChange({ serverNotice: e.target.value })}
            placeholder="e.g. Maintenance on Sunday 22:00, servers restart automatically."
          />
        )}
      </SettingsField>
    </SettingsCard>
  )
}
