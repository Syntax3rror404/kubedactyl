import { NetworkIcon, RocketIcon } from "lucide-react"

import { FloatingIcon } from "@/components/common/floating-icon"
import { RainbowLine } from "@/components/common/rainbow-line"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Spinner } from "@/components/ui/spinner"
import { StartupPreview } from "@/features/servers/components/startup-preview"
import { formatCpuLimit, formatMiB } from "@/lib/format"

/** Sticky summary of the create wizard with the startup preview and the create button. */
export function CreateSummary({
  eggName,
  name,
  memory,
  cpu,
  disk,
  ports,
  startup,
  env,
  pending,
  onCreate,
}: {
  eggName: string
  name: string
  memory: number
  cpu: number
  disk: number
  ports: number[]
  startup: string
  env: Record<string, string>
  pending: boolean
  onCreate: () => void
}) {
  return (
    <Card className="overflow-hidden">
      <RainbowLine />
      <CardHeader>
        <CardTitle>Summary</CardTitle>
        <CardDescription>{eggName}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
          <dt className="text-muted-foreground">Name</dt>
          <dd className="truncate text-right font-medium">{name}</dd>
          <dt className="text-muted-foreground">Memory</dt>
          <dd className="text-right">{formatMiB(memory)}</dd>
          <dt className="text-muted-foreground">CPU</dt>
          <dd className="text-right">{formatCpuLimit(cpu)}</dd>
          <dt className="text-muted-foreground">Disk</dt>
          <dd className="text-right">{formatMiB(disk)}</dd>
          <dt className="flex items-center gap-1 text-muted-foreground">
            <NetworkIcon className="size-3.5" />
            Ports
          </dt>
          <dd className={ports.length ? "text-right font-mono" : "text-right text-destructive"}>
            {ports.length ? ports.join(", ") : "required"}
          </dd>
        </dl>
        <Separator />
        <div className="space-y-1.5">
          <div className="text-xs font-medium text-muted-foreground">Startup command</div>
          <StartupPreview startup={startup} env={env} memory={memory} port={ports[0]} />
        </div>
        <Button className="w-full" size="lg" disabled={!ports.length || pending} onClick={onCreate}>
          {pending ? (
            <Spinner />
          ) : (
            <FloatingIcon distance={5} angle={5} duration={4}>
              <RocketIcon />
            </FloatingIcon>
          )}
          Create server
        </Button>
        {!ports.length && <p className="text-center text-xs text-muted-foreground">Enter at least one port first.</p>}
      </CardContent>
    </Card>
  )
}
