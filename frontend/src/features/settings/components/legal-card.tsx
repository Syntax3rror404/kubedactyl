import { ScaleIcon } from "lucide-react"

import { Markdown } from "@/components/common/markdown"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { SettingsCard, SettingsField } from "@/features/settings/components/settings-card"
import { tooLong, type SettingsCardProps } from "@/features/settings/lib/settings-draft"
import { legalDocs, legalTitles, type LegalDoc } from "@/lib/legal"

const MAX_LEGAL = 20000

const placeholders: Record<LegalDoc, string> = {
  legalNotice: "# Legal notice\n\nName\nStreet 1\n12345 City\n\nE-mail: admin@example.com",
  privacyPolicy: "# Privacy policy\n\n## Controller\n…\n\n## Data we process\n…",
}

/** Imprint and privacy policy (Markdown) with a live preview; linked in the footer of every page. */
export function LegalCard({ draft, onChange, errors }: SettingsCardProps) {
  return (
    <SettingsCard
      icon={ScaleIcon}
      title="Legal notice & privacy policy"
      description="Markdown. Linked in the footer of every page, including sign-in; empty texts are not shown."
    >
      <Tabs defaultValue="legalNotice">
        <TabsList>
          {legalDocs.map((d) => (
            <TabsTrigger key={d} value={d}>
              {legalTitles[d]}
            </TabsTrigger>
          ))}
        </TabsList>
        {legalDocs.map((d) => {
          const text = draft[d]
          return (
            <TabsContent key={d} value={d} className="mt-4 grid gap-4 lg:grid-cols-2">
              <SettingsField
                id={d}
                label={legalTitles[d]}
                hint={`${text.length} / ${MAX_LEGAL} characters`}
                error={errors[d] ?? tooLong(text, MAX_LEGAL)}
              >
                {(control) => (
                  <Textarea
                    {...control}
                    rows={14}
                    className="font-mono text-xs"
                    value={text}
                    placeholder={placeholders[d]}
                    onChange={(e) => onChange({ [d]: e.target.value })}
                  />
                )}
              </SettingsField>
              <div className="rounded-lg border bg-muted/30 p-4">
                <p className="mb-2 text-xs font-medium text-muted-foreground uppercase">Preview</p>
                {text.trim() ? (
                  <Markdown>{text}</Markdown>
                ) : (
                  <p className="text-sm text-muted-foreground">Nothing to show, the link stays hidden.</p>
                )}
              </div>
            </TabsContent>
          )
        })}
      </Tabs>
    </SettingsCard>
  )
}
