import { useEffect } from "react"
import { AlertTriangleIcon, ArrowUpCircleIcon, RefreshCwIcon } from "lucide-react"
import { toast } from "sonner"

import { Callout } from "@/components/common/callout"
import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { JobStateIcon } from "@/components/common/job-state-icon"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { formatRelativeTime, secondsSince } from "@/lib/format"
import { failed } from "@/lib/notify"
import { useRefreshUpgradeStatus, useLivePanelInfo, useStartUpgrade, useUpgradeStatus } from "@/lib/queries"
import type { UpgradeJob, UpgradeStatus } from "@/lib/types"

/** Newer chart versions in the registry and the button that upgrades the panel (Helm job). */
export function UpgradeCard() {
  const upgrade = useUpgradeStatus()
  const refresh = useRefreshUpgradeStatus({
    onError: failed("check for updates"),
  })
  const start = useStartUpgrade({
    onSuccess: (job) => toast.success(`Upgrade to ${job.version} started`),
    onError: failed("start the upgrade"),
  })
  const st = upgrade.data
  // The job of this panel version that is running or done: the panel restarts after it.
  const active = st?.jobs.find((j) => j.from === st.current && j.state !== "failed")
  const target = st?.newer[0]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ArrowUpCircleIcon className="size-4" />
          Panel updates
        </CardTitle>
        <CardDescription>
          {st?.enabled
            ? `The panel looks for new chart versions every ${Math.round(st.intervalSeconds / 60)} minutes. An upgrade runs helm upgrade with the values of this release and only the new version; the panel restarts and is unavailable for a moment.`
            : "Self-upgrades are off. Enable them with the chart value selfUpgrade.enabled."}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
          <span>
            Installed <span className="font-mono font-medium">v{st?.current ?? "…"}</span>
          </span>
          {st?.latest && (
            <span>
              Latest <span className="font-mono font-medium">v{st.latest}</span>
            </span>
          )}
          {st?.checkedAt && <span className="text-muted-foreground">checked {formatRelativeTime(st.checkedAt)}</span>}
          {st?.enabled && (
            <Button
              size="sm"
              variant="ghost"
              className="ml-auto"
              disabled={refresh.isPending}
              onClick={() => refresh.mutate()}
            >
              {refresh.isPending ? <Spinner /> : <RefreshCwIcon />}
              Check now
            </Button>
          )}
        </div>
        {st?.error && (
          <p className="text-sm text-destructive">
            Could not reach {st.chart}: {st.error}
          </p>
        )}

        {active ? (
          <Restarting job={active} />
        ) : (
          st?.enabled &&
          target && (
            <Callout
              tone="success"
              icon={<ArrowUpCircleIcon />}
              title={`Version ${target} is available`}
              action={
                <ConfirmDialog
                  trigger={
                    <Button disabled={start.isPending}>
                      {start.isPending ? <Spinner /> : <ArrowUpCircleIcon />}
                      Upgrade to {target}
                    </Button>
                  }
                  title={`Upgrade the panel to ${target}?`}
                  description="A job runs helm upgrade with the current values. Game servers keep running."
                  confirmLabel="Upgrade"
                  onConfirm={() => start.mutate(target)}
                />
              }
            >
              {st.newer.length > 1 && (
                <p className="text-muted-foreground">Also newer: {st.newer.slice(1).join(", ")}</p>
              )}
            </Callout>
          )
        )}
        {st?.enabled && !target && !active && !st.error && st.checkedAt && (
          <p className="text-sm text-muted-foreground">The panel is up to date.</p>
        )}

        <JobHistory status={st} />
      </CardContent>
    </Card>
  )
}

/** Shown while the job runs and the panel restarts; reloads the page once the new version answers. */
function Restarting({ job }: { job: UpgradeJob }) {
  // Errors are expected while the panel restarts; the query keeps asking.
  const live = useLivePanelInfo()
  useEffect(() => {
    if (live.data?.version === job.version) window.location.reload()
  }, [live.data?.version, job.version])
  // Still the old version long after a successful job: the values pin the image.
  if (job.state === "succeeded" && job.finishedAt && secondsSince(job.finishedAt) > 300) {
    return (
      <Callout tone="warning" icon={<AlertTriangleIcon />}>
        <p>
          The upgrade to {job.version} finished, but the panel still runs its old version. Does the release set{" "}
          <code className="font-mono">image.tag</code> in its values?
        </p>
      </Callout>
    )
  }
  return (
    <Callout
      tone="info"
      icon={<Spinner />}
      title={job.state === "running" ? `Upgrading to ${job.version}…` : `Restarting on ${job.version}…`}
      progress
    >
      <p className="text-muted-foreground">The page reloads when the new version is up.</p>
    </Callout>
  )
}

function JobHistory({ status }: { status?: UpgradeStatus }) {
  const jobs = status?.jobs.slice(0, 3) ?? []
  if (jobs.length === 0) return null
  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-muted-foreground">Recent upgrades</p>
      {jobs.map((j) => (
        <div key={j.name} className="space-y-2 rounded-lg border p-3 text-sm">
          <div className="flex items-center gap-2">
            <JobStateIcon state={j.state} />
            <span className="font-mono">
              {j.from} → {j.version}
            </span>
            <Badge variant={j.state === "failed" ? "destructive" : "secondary"}>{j.state}</Badge>
            <span className="ml-auto text-xs text-muted-foreground">
              {formatRelativeTime(j.finishedAt ?? j.startedAt)}
            </span>
          </div>
          {j.log && (
            <pre className="max-h-40 overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap">
              {j.log}
            </pre>
          )}
        </div>
      ))}
    </div>
  )
}
