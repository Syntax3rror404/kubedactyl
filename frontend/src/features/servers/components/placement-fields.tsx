import { AlertTriangleIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { formatPoolUsage, poolUsage } from "@/lib/format"
import type { Pool, TrafficPolicy } from "@/lib/types"

/**
 * Dropdown of the storage classes or load balancer pools enabled in the panel settings.
 * The current value stays listed even when it was disabled later.
 */
export function PlacementSelect({
  id,
  value,
  options,
  defaultName,
  onChange,
}: {
  id: string
  value: string
  options: string[]
  defaultName?: string
  onChange: (value: string) => void
}) {
  const items = value && !options.includes(value) ? [value, ...options] : options
  return (
    <Select value={value} onValueChange={onChange} disabled={items.length === 0}>
      <SelectTrigger id={id} className="w-full font-mono">
        <SelectValue placeholder="none enabled" />
      </SelectTrigger>
      <SelectContent>
        {items.map((o) => (
          <SelectItem key={o} value={o} className="font-mono">
            {o}
            {o === defaultName && (
              <Badge variant="secondary" className="ml-1 font-sans text-[10px]">
                default
              </Badge>
            )}
            {!options.includes(o) && (
              <span className="font-sans text-xs text-muted-foreground">(no longer enabled)</span>
            )}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

/** Load balancer pool selection with the pool's addresses (admins get pool details). */
export function PoolField({
  value,
  options,
  defaultName,
  onChange,
  pools,
  hint,
}: {
  value: string
  options: string[]
  defaultName?: string
  onChange: (value: string) => void
  pools?: Pool[]
  hint?: string
}) {
  const pool = pools?.find((p) => p.name === value)
  return (
    <Field>
      <FieldLabel htmlFor="pool">Load balancer pool</FieldLabel>
      <PlacementSelect id="pool" value={value} options={options} defaultName={defaultName} onChange={onChange} />
      <FieldDescription>
        {pool ? [pool.blocks?.join(", "), ...poolUsage(pool).map(formatPoolUsage)].filter(Boolean).join(" · ") : hint}
      </FieldDescription>
    </Field>
  )
}

const trafficPolicies: { value: TrafficPolicy; description: string }[] = [
  { value: "Local", description: "Routes directly to the workload node (Preserve Client Source IP)." },
  { value: "Cluster", description: "Routes over the nodes via SNAT." },
]

/** External traffic policy of the server's load balancer service (Local unless set). */
export function TrafficPolicyField({
  value,
  onChange,
}: {
  value: TrafficPolicy
  onChange: (value: TrafficPolicy) => void
}) {
  return (
    <Field>
      <FieldLabel htmlFor="traffic-policy">External traffic policy</FieldLabel>
      <Select value={value} onValueChange={(v) => onChange(v as TrafficPolicy)}>
        <SelectTrigger id="traffic-policy" className="w-full font-mono">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {trafficPolicies.map((p) => (
            <SelectItem key={p.value} value={p.value} className="font-mono">
              {p.value}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <FieldDescription>{trafficPolicies.find((p) => p.value === value)?.description}</FieldDescription>
    </Field>
  )
}

/** Fixed IPs without the IPv6 ones ("192.0.2.170,2001:db8::5" -> "192.0.2.170"). */
function ipv4Only(fixedIPs: string) {
  return fixedIPs
    .split(",")
    .map((ip) => ip.trim())
    .filter((ip) => ip && !ip.includes(":"))
    .join(",")
}

/**
 * Asks for an IPv6 address too (PreferDualStack). Switching it off also drops a fixed IPv6 address:
 * the server would not get it any more.
 */
export function IPv6Field({
  checked,
  fixedIPs,
  addresses,
  unavailable,
  onChange,
}: {
  checked: boolean
  fixedIPs: string
  /** Addresses of the saved server with IPv6 on, for the hint when it got none. */
  addresses?: string[]
  /** Why the cluster gives services no IPv6 address (known before saving). */
  unavailable?: string
  onChange: (ipv6: boolean, fixedIPs: string) => void
}) {
  const missing = checked && !!addresses?.length && !addresses.some((a) => a.includes(":"))
  const warning = checked && unavailable ? "IPv6 not available in cluster." : missing && "no IPv6 address assigned."
  return (
    <Field orientation="horizontal">
      <Switch id="ipv6" checked={checked} onCheckedChange={(v) => onChange(v, v ? fixedIPs : ipv4Only(fixedIPs))} />
      <div>
        <FieldLabel htmlFor="ipv6">
          IPv6
          <span className="font-mono text-xs font-normal text-muted-foreground">
            {checked ? "PreferDualStack" : "SingleStack"}
          </span>
        </FieldLabel>
        <FieldDescription>
          {!checked ? (
            "Uses the first IP family of the cluster's service CIDRs."
          ) : warning ? (
            <span className="inline-flex items-center gap-1 text-amber-500" title={unavailable}>
              <AlertTriangleIcon className="size-3.5" />
              Warning: {warning}
            </span>
          ) : (
            "Also an IPv6 address if the cluster runs dual stack and the pool has an IPv6 block. The game must listen on IPv6."
          )}
        </FieldDescription>
      </div>
    </Field>
  )
}
