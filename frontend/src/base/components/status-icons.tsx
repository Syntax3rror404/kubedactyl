import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  CircleSlashIcon,
  InfoIcon,
  MinusCircleIcon,
  XCircleIcon,
} from "lucide-react"

import type { CheckState, JobState } from "@/base/types"
import { Spinner } from "@/components/ui/spinner"
import { cn } from "@/lib/utils"

const icons = {
  ok: [CheckCircle2Icon, "text-success"],
  warning: [AlertTriangleIcon, "text-warning"],
  error: [XCircleIcon, "text-destructive"],
  info: [InfoIcon, "text-info"],
  skipped: [MinusCircleIcon, "text-muted-foreground"],
} as const

/** The icon of a check result, the same on every page with checks. */
export function CheckStatusIcon({ status, className }: { status: CheckState; className?: string }) {
  const [Icon, color] = icons[status]
  return <Icon className={cn("shrink-0", color, className)} aria-label={status} />
}

/** Spinner while a background job runs, then a check mark, a cross or (cancelled) a slashed circle. */
export function JobStateIcon({ state }: { state: JobState }) {
  if (state === "running") return <Spinner className="size-4 shrink-0 text-info" />
  if (state === "failed") return <XCircleIcon className="size-4 shrink-0 text-destructive" />
  if (state === "cancelled") return <CircleSlashIcon className="size-4 shrink-0 text-muted-foreground" />
  return <CheckCircle2Icon className="size-4 shrink-0 text-success" />
}
