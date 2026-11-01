import { GaugeIcon } from "lucide-react"

import { FieldGroup } from "@/components/ui/field"
import { Slider } from "@/components/ui/slider"
import { SettingsCard, SettingsField } from "@/features/settings/components/settings-card"
import type { SettingsCardProps } from "@/features/settings/lib/settings-draft"

/** How many requests per second the panel sends to the Kubernetes API, and one user to the panel. */
export function KubeApiCard({ draft, onChange, errors }: SettingsCardProps) {
  return (
    <SettingsCard icon={GaugeIcon} title="Kube API limit" description="Protects the Kubernetes API from overload.">
      <FieldGroup className="gap-6">
        <LimitField
          id="kube-api-qps"
          label="Requests per second"
          value={draft.kubeApiQps}
          range={[5, 1000, 5]}
          onChange={(kubeApiQps) => onChange({ kubeApiQps })}
          error={errors.kubeApiQps}
          hint="Default: 50"
        />
        <LimitField
          id="kube-api-user-qps"
          label="Per user"
          value={draft.kubeApiUserQps}
          range={[1, 200, 1]}
          onChange={(kubeApiUserQps) => onChange({ kubeApiUserQps })}
          error={errors.kubeApiUserQps}
          hint="So one user cannot slow down everybody else. Default: 10"
        />
      </FieldGroup>
    </SettingsCard>
  )
}

/** A slider with its value next to the label. range = [min, max, step]. */
function LimitField({
  id,
  label,
  value,
  range: [min, max, step],
  onChange,
  error,
  hint,
}: {
  id: string
  label: string
  value: number
  range: [number, number, number]
  onChange: (value: number) => void
  error?: string
  hint: string
}) {
  return (
    <SettingsField
      id={id}
      label={label}
      aside={<span className="rounded-md bg-muted px-2 py-0.5 font-mono text-sm tabular-nums">{value}</span>}
      hint={hint}
      error={error}
    >
      {(control) => (
        <Slider {...control} value={[value]} min={min} max={max} step={step} onValueChange={(v) => onChange(v[0])} />
      )}
    </SettingsField>
  )
}
