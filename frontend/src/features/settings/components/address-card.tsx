import { ArrowRightIcon, GlobeIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldLabel, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

/** External domain that users see instead of the load balancer IP. */
export function AddressCard({
  domain,
  onChange,
  exampleIP,
  error,
}: {
  domain: string
  onChange: (domain: string) => void
  exampleIP?: string
  error?: string
}) {
  const ip = exampleIP ?? "192.168.1.61"
  const shown = domain.trim().toLowerCase().replace(/\.$/, "")
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GlobeIcon className="size-4" />
          Server address
        </CardTitle>
        <CardDescription>
          Users see this domain instead of the load balancer IP. Administrators still see the IP.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Field data-invalid={error ? true : undefined}>
          <FieldLabel htmlFor="domain">External domain</FieldLabel>
          <Input
            id="domain"
            value={domain}
            onChange={(e) => onChange(e.target.value)}
            placeholder="play.example.com"
            className="font-mono"
            aria-invalid={error ? true : undefined}
          />
          {error ? (
            <FieldError>{error}</FieldError>
          ) : (
            <FieldDescription>Must point to the IPs of the enabled pools. Empty shows the IP.</FieldDescription>
          )}
        </Field>
        <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-muted/40 px-3 py-2 font-mono text-sm">
          <span className={shown ? "text-muted-foreground line-through" : undefined}>{ip}:25565</span>
          {shown && (
            <>
              <ArrowRightIcon className="size-4 text-muted-foreground" />
              <span className="font-medium">{shown}:25565</span>
            </>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
