import { MegaphoneIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Textarea } from "@/components/ui/textarea"

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
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MegaphoneIcon className="size-4" />
          Server notice
        </CardTitle>
        <CardDescription>
          Shown as a dialog whenever users open one of their servers. Empty shows nothing.
        </CardDescription>
      </CardHeader>
      <CardContent>
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
      </CardContent>
    </Card>
  )
}
