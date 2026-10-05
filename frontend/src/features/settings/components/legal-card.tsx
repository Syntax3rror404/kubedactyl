import { ScaleIcon } from "lucide-react"

import { Markdown } from "@/components/common/markdown"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { legalDocs, legalTitles, type LegalDoc } from "@/lib/legal"

const MAX = 20000

const placeholders: Record<LegalDoc, string> = {
  legalNotice: "# Legal notice\n\nName\nStreet 1\n12345 City\n\nE-mail: admin@example.com",
  privacyPolicy: "# Privacy policy\n\n## Controller\n…\n\n## Data we process\n…",
}

/** Imprint and privacy policy (Markdown) with a live preview; linked in the footer of every page. */
export function LegalCard({
  values,
  onChange,
  errors,
}: {
  values: Record<LegalDoc, string>
  onChange: (key: LegalDoc, text: string) => void
  errors: Record<string, string>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ScaleIcon className="size-4" />
          Legal notice & privacy policy
        </CardTitle>
        <CardDescription>
          Markdown. Linked in the footer of every page, including sign-in; empty texts are not shown.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="legalNotice">
          <TabsList>
            {legalDocs.map((d) => (
              <TabsTrigger key={d} value={d}>
                {legalTitles[d]}
              </TabsTrigger>
            ))}
          </TabsList>
          {legalDocs.map((d) => {
            const text = values[d]
            const error =
              errors[d] ?? (text.length > MAX ? `Too long (${text.length} of ${MAX} characters)` : undefined)
            return (
              <TabsContent key={d} value={d} className="mt-4 grid gap-4 lg:grid-cols-2">
                <Field data-invalid={!!error}>
                  <FieldLabel htmlFor={d}>{legalTitles[d]}</FieldLabel>
                  <Textarea
                    id={d}
                    rows={14}
                    className="font-mono text-xs"
                    value={text}
                    placeholder={placeholders[d]}
                    onChange={(e) => onChange(d, e.target.value)}
                    aria-invalid={!!error}
                  />
                  {error ? (
                    <FieldError>{error}</FieldError>
                  ) : (
                    <FieldDescription>
                      {text.length} / {MAX} characters
                    </FieldDescription>
                  )}
                </Field>
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
      </CardContent>
    </Card>
  )
}
