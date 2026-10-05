import { CheckCircle2Icon, XCircleIcon } from "lucide-react"

import { Spinner } from "@/components/ui/spinner"
import type { ServerJob, UpgradeJob } from "@/lib/types"

/** Spinner while a background job runs, then a check mark or a cross. */
export function JobStateIcon({ state }: { state: ServerJob["state"] | UpgradeJob["state"] }) {
  if (state === "running") return <Spinner className="size-4 shrink-0 text-sky-500" />
  if (state === "failed") return <XCircleIcon className="size-4 shrink-0 text-destructive" />
  return <CheckCircle2Icon className="size-4 shrink-0 text-emerald-500" />
}
