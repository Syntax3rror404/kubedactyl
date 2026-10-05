import { ShieldIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"

/** Egress isolation of user namespaces (NetworkPolicy kubedactyl-isolation). */
export function IsolationCard({ isolated, onChange }: { isolated: boolean; onChange: (isolated: boolean) => void }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ShieldIcon className="size-4" />
          Network isolation
        </CardTitle>
        <CardDescription>
          Isolated servers only reach the internet, the cluster DNS and their owner's other servers.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Field orientation="horizontal">
          <Switch id="isolation" checked={isolated} onCheckedChange={onChange} />
          <div>
            <FieldLabel htmlFor="isolation">Isolate user namespaces (recommended)</FieldLabel>
            <FieldDescription>
              Blocks other namespaces, nodes, the Kubernetes API and private networks. Players are not affected.
            </FieldDescription>
          </div>
        </Field>
      </CardContent>
    </Card>
  )
}
