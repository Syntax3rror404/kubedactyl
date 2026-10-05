import { useNavigate } from "react-router"
import { toast } from "sonner"

import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { EggOption } from "@/features/eggs/components/egg-option"
import { CreateSummary } from "@/features/servers/components/create-summary"
import { FormSection } from "@/features/servers/components/form-section"
import { OwnerSelect } from "@/features/servers/components/owner-select"
import { PlacementSelect, PoolField, TrafficPolicyField } from "@/features/servers/components/placement-fields"
import { PortEditor } from "@/features/servers/components/port-editor"
import { ResourceSliders } from "@/features/servers/components/resource-sliders"
import { VariableField } from "@/features/servers/components/variable-field"
import type { SharedServerFields } from "@/features/servers/hooks/use-shared-server-fields"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useCreateServer, usePools, useSettings } from "@/lib/queries"
import type { Egg } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/** The create form for one egg: image, variables, resources, ports, placement and owner. */
export function ServerForm({
  egg,
  eggs,
  onSelectEgg,
  shared,
}: {
  egg: Egg
  eggs: Egg[]
  onSelectEgg: (name: string) => void
  shared: SharedServerFields
}) {
  const settings = useSettings()
  const pools = usePools()
  const navigate = useNavigate()
  const f = shared.draft
  // Egg specific fields; the parent mounts a new form for every egg.
  const own = useDraft({
    image: egg.spec.dockerImages[0]?.image ?? "",
    env: Object.fromEntries((egg.spec.variables ?? []).map((v) => [v.envVariable, v.defaultValue ?? ""])),
  })
  const { image, env } = own.draft
  const displayName = f.name.trim() || egg.spec.displayName
  const storageClass = f.storageClass || settings.data?.defaultStorageClass || ""
  const pool = f.pool || settings.data?.defaultLoadBalancerPool || ""
  const domain = settings.data?.externalDomain

  const create = useCreateServer({
    onSuccess: (gs) => {
      toast.success(`Server “${gs.spec.displayName}” created`, { description: "The install script is running now." })
      navigate(`/servers/${gs.metadata.name}`)
    },
    onError: failed("create the server"),
  })
  const errors = fieldErrors(create.error)

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_22rem]">
      <div className="space-y-6">
        <FormSection step={1} title="Egg" description="The template that defines image, install script and variables.">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {eggs.map((e) => (
              <EggOption
                key={e.metadata.uid}
                egg={e}
                selected={e.metadata.name === egg.metadata.name}
                onSelect={() => onSelectEgg(e.metadata.name)}
              />
            ))}
          </div>
        </FormSection>

        <FormSection step={2} title="General">
          <FieldGroup className="grid gap-5 sm:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="name">Server name</FieldLabel>
              <Input
                id="name"
                value={f.name}
                onChange={(e) => shared.set("name", e.target.value)}
                placeholder={egg.spec.displayName}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="image">Docker image</FieldLabel>
              <Select value={image} onValueChange={(v) => own.set("image", v)}>
                <SelectTrigger id="image" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {egg.spec.dockerImages.map((img) => (
                    <SelectItem key={img.image} value={img.image}>
                      {img.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldDescription className="truncate font-mono text-xs">{image}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="owner">Owner</FieldLabel>
              <OwnerSelect id="owner" value={f.owner} onChange={(v) => shared.set("owner", v)} />
            </Field>
            <Field orientation="horizontal" className="self-center">
              <Switch id="start" checked={f.start} onCheckedChange={(v) => shared.set("start", v)} />
              <FieldLabel htmlFor="start">Start the server when the installation has finished</FieldLabel>
            </Field>
          </FieldGroup>
        </FormSection>

        <FormSection
          step={3}
          title="Resources"
          description="Memory is exposed as SERVER_MEMORY; the container limit adds a small overhead for the runtime."
        >
          <ResourceSliders
            memory={f.memory}
            onMemoryChange={(v) => shared.set("memory", v)}
            cpu={f.cpu}
            onCpuChange={(v) => shared.set("cpu", v)}
            disk={f.disk}
            onDiskChange={(v) => shared.set("disk", v)}
            diskHint={`Volume of storage class ${storageClass || "…"}, can be grown later`}
          />
          <Field className="mt-6 sm:max-w-xs">
            <FieldLabel htmlFor="storage">Storage class</FieldLabel>
            <PlacementSelect
              id="storage"
              value={storageClass}
              options={settings.data?.storageClasses ?? []}
              defaultName={settings.data?.defaultStorageClass}
              onChange={(v) => shared.set("storageClass", v)}
            />
            <FieldDescription>Fixed for the life of the volume.</FieldDescription>
          </Field>
        </FormSection>

        <FormSection
          step={4}
          title="Network"
          description={`Every port is published as TCP and UDP on a load balancer IP of the selected pool.${domain ? ` Users connect via ${domain}:${f.ports[0] ?? ""}.` : ""}`}
        >
          <FieldGroup className="grid gap-5 sm:grid-cols-2">
            <Field data-invalid={f.ports.length === 0}>
              <FieldLabel htmlFor="ports">
                Ports <span className="text-destructive">*</span>
              </FieldLabel>
              <PortEditor id="ports" ports={f.ports} onChange={(v) => shared.set("ports", v)} />
              {f.ports.length === 0 ? (
                <FieldError>At least one port is required. The first one becomes SERVER_PORT.</FieldError>
              ) : (
                <FieldDescription>The first port becomes SERVER_PORT.</FieldDescription>
              )}
            </Field>
            <PoolField
              value={pool}
              options={settings.data?.loadBalancerPools ?? []}
              defaultName={settings.data?.defaultLoadBalancerPool}
              onChange={(v) => shared.set("pool", v)}
              pools={pools.data?.items}
            />
            <Field>
              <FieldLabel htmlFor="lbip">Fixed IP (optional)</FieldLabel>
              <Input
                id="lbip"
                value={f.lbIP}
                onChange={(e) => shared.set("lbIP", e.target.value)}
                placeholder="assigned automatically"
                className="font-mono"
              />
              <FieldDescription>Must be a free address of the selected pool.</FieldDescription>
            </Field>
            <TrafficPolicyField value={f.trafficPolicy} onChange={(v) => shared.set("trafficPolicy", v)} />
          </FieldGroup>
        </FormSection>

        {(egg.spec.variables?.length ?? 0) > 0 && (
          <FormSection step={5} title="Variables" description="Validated against the rules of the egg.">
            <FieldGroup className="grid gap-6 md:grid-cols-2">
              {egg.spec.variables!.map((v) => (
                <VariableField
                  key={v.envVariable}
                  variable={v}
                  value={env[v.envVariable] ?? ""}
                  error={errors[v.envVariable]}
                  onChange={(val) => own.set("env", { ...env, [v.envVariable]: val })}
                />
              ))}
            </FieldGroup>
          </FormSection>
        )}
      </div>

      <div className="lg:sticky lg:top-6 lg:self-start">
        <CreateSummary
          eggName={egg.spec.displayName}
          name={displayName}
          memory={f.memory}
          cpu={f.cpu}
          disk={f.disk}
          ports={f.ports}
          startup={egg.spec.startup}
          env={env}
          pending={create.isPending}
          onCreate={() =>
            create.mutate({
              displayName,
              egg: egg.metadata.name,
              image,
              environment: env,
              memoryMiB: f.memory,
              cpuMillis: f.cpu,
              diskMiB: f.disk,
              ports: f.ports,
              storageClass: storageClass || undefined,
              loadBalancerPool: pool || undefined,
              loadBalancerIP: f.lbIP.trim() || undefined,
              externalTrafficPolicy: f.trafficPolicy,
              owner: f.owner,
              startOnCompletion: f.start,
            })
          }
        />
      </div>
    </div>
  )
}
