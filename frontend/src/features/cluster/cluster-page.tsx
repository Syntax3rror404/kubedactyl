import { RefreshCwIcon } from "lucide-react"
import { toast } from "sonner"

import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { ClusterSummary } from "@/features/cluster/components/cluster-summary"
import { AuthCard, ConnectionCard, PermissionsCard } from "@/features/cluster/components/connection-cards"
import { NodeCard } from "@/features/cluster/components/node-card"
import { formatRelativeTime } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useClusterIdentity, useClusterNodes, useProbeNodes } from "@/lib/queries"

/** /cluster (admins): nodes with hardware and usage, totals, connection, identity and permissions. */
export function ClusterPage() {
  const nodes = useClusterNodes()
  const identity = useClusterIdentity()
  const probe = useProbeNodes({
    onSuccess: (data) => {
      toast.success("Hardware probed", { description: `${data.nodes.filter((n) => n.hardware).length} nodes probed` })
    },
    onError: failed("probe the hardware"),
  })
  const probedAt = nodes.data?.nodes
    .map((n) => n.hardwareProbedAt)
    .filter(Boolean)
    .sort()
    .at(-1)

  return (
    <div className="space-y-8">
      <PageHeader
        title="Cluster"
        description={
          identity.data
            ? `Kubernetes ${identity.data.version} · ${identity.data.server}`
            : "Nodes, hardware and the identity of the panel."
        }
        actions={
          <Button
            variant="outline"
            disabled={probe.isPending}
            onClick={() => probe.mutate()}
            title="Starts a short-lived probe pod on every ready node"
          >
            {probe.isPending ? <Spinner /> : <RefreshCwIcon />}
            Re-read hardware{probedAt && !probe.isPending ? ` · ${formatRelativeTime(probedAt)}` : ""}
          </Button>
        }
      />

      <QueryState query={nodes}>
        {(data) => (
          <>
            <ClusterSummary totals={data.totals} />
            <section className="space-y-4">
              <h2 className="text-lg font-semibold tracking-tight">Nodes</h2>
              <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
                {data.nodes.map((n) => (
                  <NodeCard key={n.name} node={n} />
                ))}
              </div>
            </section>
          </>
        )}
      </QueryState>

      <section className="space-y-4">
        <h2 className="text-lg font-semibold tracking-tight">Access</h2>
        <QueryState query={identity}>
          {(id) => (
            <div className="grid gap-5 lg:grid-cols-2 xl:grid-cols-3">
              <ConnectionCard id={id} />
              <AuthCard id={id} />
              <PermissionsCard id={id} />
            </div>
          )}
        </QueryState>
      </section>
    </div>
  )
}
