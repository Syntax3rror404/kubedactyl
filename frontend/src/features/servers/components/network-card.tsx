import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { IPv6Field, PoolField, TrafficPolicyField } from "@/features/servers/components/placement-fields"
import { PortEditor } from "@/features/servers/components/port-editor"
import type { SettingsChange, SettingsDraft } from "@/features/servers/lib/settings-draft"
import { serverAddress } from "@/lib/format"
import { usePools, useSettings } from "@/lib/queries"
import type { GameServer } from "@/lib/types"

/**
 * Settings page: the load balancer pool; administrators also set the ports, fixed IPs, the
 * traffic policy and IPv6. Users only see it when pools are configured.
 */
export function NetworkCard({
  server,
  draft,
  onChange,
  errors,
  isAdmin,
}: {
  server: GameServer
  draft: SettingsDraft
  onChange: SettingsChange
  errors: Record<string, string>
  isAdmin: boolean
}) {
  const settings = useSettings().data
  const pools = usePools(isAdmin)
  const address = (server.status?.addresses ?? [server.status?.address]).filter(Boolean).join(", ") || undefined
  const saved = server.spec.loadBalancerPool ?? ""
  if (!isAdmin && !settings?.loadBalancerPools?.length) return null
  const poolField = (
    <PoolField
      value={draft.pool}
      options={settings?.loadBalancerPools ?? []}
      defaultName={settings?.defaultLoadBalancerPool}
      // A fixed IP belongs to the previous pool.
      onChange={(pool) => onChange(pool === saved ? { pool } : { pool, lbIP: "" })}
      pools={pools.data?.items}
      hint={
        draft.pool !== saved
          ? "The server gets a new address from this pool when you save."
          : "Moving the server to another pool changes its address."
      }
    />
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle>Network</CardTitle>
        <CardDescription>
          {isAdmin
            ? "Ports are published as TCP and UDP. The first one is SERVER_PORT."
            : "The pool your server's address comes from."}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isAdmin ? (
          <FieldGroup>
            <Field data-invalid={draft.ports.length === 0 || !!errors.ports}>
              <FieldLabel htmlFor="ports">Ports</FieldLabel>
              <PortEditor id="ports" ports={draft.ports} onChange={(ports) => onChange({ ports })} />
              {draft.ports.length === 0 && <FieldError>At least one port is required.</FieldError>}
              {errors.ports && <FieldError>{errors.ports}</FieldError>}
            </Field>
            {poolField}
            <Field data-invalid={!!errors.loadBalancerIP}>
              <FieldLabel htmlFor="ip">Fixed load balancer IP</FieldLabel>
              <Input
                id="ip"
                value={draft.lbIP}
                onChange={(e) => onChange({ lbIP: e.target.value })}
                placeholder={address ?? "automatic"}
                className="font-mono"
              />
              <FieldDescription>
                Current address: {address ?? "pending"}
                {settings?.externalDomain && address && `, users see ${serverAddress(server, settings.externalDomain)}`}
              </FieldDescription>
              {errors.loadBalancerIP && <FieldError>{errors.loadBalancerIP}</FieldError>}
            </Field>
            <TrafficPolicyField value={draft.trafficPolicy} onChange={(trafficPolicy) => onChange({ trafficPolicy })} />
            <IPv6Field
              checked={draft.ipv6}
              fixedIPs={draft.lbIP}
              addresses={server.spec.ipv6 ? server.status?.addresses : undefined}
              unavailable={pools.data?.ipv6Missing}
              onChange={(ipv6, lbIP) => onChange({ ipv6, lbIP })}
            />
          </FieldGroup>
        ) : (
          <FieldGroup>{poolField}</FieldGroup>
        )}
      </CardContent>
    </Card>
  )
}
