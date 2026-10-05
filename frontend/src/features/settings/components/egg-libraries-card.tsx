import { LibraryIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Textarea } from "@/components/ui/textarea"

/** The GitHub repositories the egg library lists (one URL per line). */
export function EggLibrariesCard({
  repositories,
  onChange,
  error,
}: {
  repositories: string[]
  onChange: (repositories: string[]) => void
  error?: string
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <LibraryIcon className="size-4" />
          Egg library
        </CardTitle>
        <CardDescription>GitHub repositories whose eggs the library offers to install.</CardDescription>
      </CardHeader>
      <CardContent>
        <Field data-invalid={error ? true : undefined}>
          <FieldLabel htmlFor="egg-libraries">Repositories</FieldLabel>
          <Textarea
            id="egg-libraries"
            rows={4}
            className="font-mono text-sm"
            value={repositories.join("\n")}
            onChange={(e) => onChange(e.target.value.split("\n"))}
            aria-invalid={error ? true : undefined}
          />
          {error ? (
            <FieldError>{error}</FieldError>
          ) : (
            <FieldDescription>
              One repository per line (https://github.com/&lt;owner&gt;/&lt;repo&gt;).
            </FieldDescription>
          )}
        </Field>
      </CardContent>
    </Card>
  )
}
