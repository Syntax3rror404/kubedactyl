import { rateItems } from "@/app/config"
import { useRequestRates } from "@/base/lib/queries"
import type { Rate } from "@/base/lib/types"
import type { RateItem as RateConfig } from "@/base/types"
import { cn } from "@/lib/utils"

/** Lamp color by how much of the limit is used: green below half, yellow below 90 %, red from there on. */
function lamp({ rate, limit }: Rate) {
  const used = limit > 0 ? rate / limit : 0
  if (used < 0.5) return "bg-success"
  if (used < 0.9) return "bg-warning"
  return "bg-destructive"
}

function RateItem({ item, rate, labelled }: { item: RateConfig; rate: Rate; labelled: boolean }) {
  const Icon = item.icon
  return (
    <span className="inline-flex items-center gap-1 tabular-nums [&_svg]:size-3.5" title={item.title}>
      <Icon />
      {labelled && `${item.label} `}
      {rate.rate.toFixed(1)}/{rate.limit} req/s
      <span aria-hidden className={cn("ml-0.5 inline-block size-2 shrink-0 rounded-full", lamp(rate))} />
    </span>
  )
}

/**
 * Footer: the rates of app/config.ts against their limits, so a limit is seen coming: the user's own requests and,
 * for administrators, the calls of the whole app. Labels show only when there is more than one rate.
 */
export function RequestRates() {
  const { data } = useRequestRates()
  if (!data) return null
  const rates = new Map(Object.entries(data))
  const shown = rateItems.flatMap((item) => {
    const rate = rates.get(item.key)
    return rate ? [{ item, rate }] : []
  })
  return shown.map(({ item, rate }) => <RateItem key={item.key} item={item} rate={rate} labelled={shown.length > 1} />)
}
