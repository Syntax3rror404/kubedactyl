import { Badge } from "@/components/ui/badge"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
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
        {pool
          ? `${pool.blocks?.join(", ") ?? ""}${pool.ipsTotal >= 0 ? ` · ${pool.ipsAvailable} of ${pool.ipsTotal} free` : ""}`
          : hint}
      </FieldDescription>
    </Field>
  )
}

const trafficPolicies: { value: TrafficPolicy; description: string }[] = [
  {
    value: "Local",
    description: "The server sees the players' IP addresses (ban lists, logs). Only the node running it answers.",
  },
  {
    value: "Cluster",
    description: "Every node answers and forwards the traffic; the server sees a node address instead of the player's.",
  },
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
