import { CheckCircle2Icon, CircleSlashIcon, XCircleIcon } from "lucide-react"

import { Spinner } from "@/components/ui/spinner"
import type { ServerJob, UpgradeJob } from "@/lib/types"

/** Spinner while a background job runs, then a check mark, a cross or (cancelled) a slashed circle. */
export function JobStateIcon({ state }: { state: ServerJob["state"] | UpgradeJob["state"] }) {
  if (state === "running") return <Spinner className="size-4 shrink-0 text-sky-500" />
  if (state === "failed") return <XCircleIcon className="size-4 shrink-0 text-destructive" />
  if (state === "cancelled") return <CircleSlashIcon className="size-4 shrink-0 text-muted-foreground" />
  return <CheckCircle2Icon className="size-4 shrink-0 text-emerald-500" />
}
