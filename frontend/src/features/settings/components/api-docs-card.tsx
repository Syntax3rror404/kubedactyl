import { BookOpenIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"

/** Whether the API documentation (Swagger UI at /swagger/) is served. */
export function ApiDocsCard({ enabled, onChange }: { enabled: boolean; onChange: (enabled: boolean) => void }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BookOpenIcon className="size-4" />
          API documentation
        </CardTitle>
        <CardDescription>Interactive REST API documentation, linked in the sidebar.</CardDescription>
      </CardHeader>
      <CardContent>
        <Field orientation="horizontal">
          <Switch id="api-docs" checked={enabled} onCheckedChange={onChange} />
          <div>
            <FieldLabel htmlFor="api-docs">Serve the API documentation (Swagger UI)</FieldLabel>
            <FieldDescription>Turned off, /swagger/ answers “not found”. The API keeps working.</FieldDescription>
          </div>
        </Field>
      </CardContent>
    </Card>
  )
}
