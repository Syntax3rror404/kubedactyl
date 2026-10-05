import { CopyButton } from "@/components/common/copy-button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { GameServer } from "@/lib/types"

/** kubectl commands for the objects the controller manages for a server. */
export function KubernetesCard({ server }: { server: GameServer }) {
  const name = server.metadata.name
  const ns = server.metadata.namespace
  const commands = [
    ["Game server", `kubectl -n ${ns} get gameservers.kubedactyl.io ${name}`],
    ["Pods", `kubectl -n ${ns} get pods -l kubedactyl.io/server=${name}`],
    ["Logs", `kubectl -n ${ns} logs -f ${name}-game`],
    ["Volume", `kubectl -n ${ns} get pvc ${name}-data`],
    ["Service", `kubectl -n ${ns} get svc ${name}`],
  ]
  return (
    <Card>
      <CardHeader>
        <CardTitle>Kubernetes</CardTitle>
        <CardDescription>Objects managed for this server.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {commands.map(([label, cmd]) => (
          <div key={label} className="flex items-center gap-2 rounded-lg border bg-muted/40 py-1 pr-1 pl-3">
            <span className="w-20 shrink-0 text-xs text-muted-foreground">{label}</span>
            <code className="flex-1 truncate font-mono text-xs">{cmd}</code>
            <CopyButton value={cmd} />
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
