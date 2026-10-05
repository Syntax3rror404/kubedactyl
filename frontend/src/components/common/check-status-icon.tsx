import { AlertTriangleIcon, CheckCircle2Icon, MinusCircleIcon, XCircleIcon } from "lucide-react"

import type { CheckStatus } from "@/lib/types"
import { cn } from "@/lib/utils"

const icons = {
  ok: [CheckCircle2Icon, "text-emerald-500"],
  warning: [AlertTriangleIcon, "text-amber-500"],
  error: [XCircleIcon, "text-red-500"],
  skipped: [MinusCircleIcon, "text-muted-foreground"],
} as const

/** The icon of a check result, the same in the cluster card and the server diagnostics. */
export function CheckStatusIcon({ status, className }: { status: CheckStatus; className?: string }) {
  const [Icon, color] = icons[status]
  return <Icon className={cn("shrink-0", color, className)} aria-label={status} />
}
