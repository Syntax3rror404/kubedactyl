import { ArrowDownUpIcon, ServerPlusIcon, type LucideIcon } from "lucide-react"

import { useRequestRates } from "@/lib/queries"
import type { Rate } from "@/lib/types"
import { cn } from "@/lib/utils"

/** Lamp color by how much of the limit is used: green below half, yellow below 90 %, red from there on. */
function lamp({ rate, limit }: Rate) {
  const used = limit > 0 ? rate / limit : 0
  if (used < 0.5) return "bg-emerald-500"
  if (used < 0.9) return "bg-amber-500"
  return "bg-red-500"
}

function RateItem({ icon: Icon, label, rate, title }: { icon: LucideIcon; label?: string; rate: Rate; title: string }) {
  return (
    <span className="inline-flex items-center gap-1 tabular-nums [&_svg]:size-3.5" title={title}>
      <Icon />
      {label && `${label} `}
      {rate.rate.toFixed(1)}/{rate.limit} req/s
      <span aria-hidden className={cn("ml-0.5 inline-block size-2 shrink-0 rounded-full", lamp(rate))} />
    </span>
  )
}

/**
 * Footer: the user's requests per second against the per-user limit and, for admins, the Kubernetes API calls of
 * the whole panel against the panel limit (settings "Kube API limit"), so a limit is seen coming.
 */
export function RequestRates() {
  const { data } = useRequestRates()
  if (!data) return null
  const panel = data.panel
  return (
    <>
      <RateItem
        icon={ArrowDownUpIcon}
        label={panel ? "User" : undefined}
        rate={data.user}
        title="Your requests per second over the last 5 seconds and the limit per user"
      />
      {panel && (
        <RateItem
          icon={ServerPlusIcon}
          label="System"
          rate={panel}
          title="Kubernetes API calls per second of the whole panel over the last 5 seconds and the panel limit"
        />
      )}
    </>
  )
}
