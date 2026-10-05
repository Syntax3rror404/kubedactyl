import { GaugeIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Slider } from "@/components/ui/slider"

type Limits = { kubeApiQps: number; kubeApiUserQps: number }

/** How many requests per second the panel sends to the Kubernetes API, and one user to the panel. */
export function KubeApiCard({
  values,
  onChange,
  errors,
}: {
  values: Limits
  onChange: (patch: Partial<Limits>) => void
  errors: Record<string, string>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GaugeIcon className="size-4" />
          Kube API limit
        </CardTitle>
        <CardDescription>Protects the Kubernetes API from overload.</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-6">
          <LimitField
            id="kube-api-qps"
            label="Requests per second"
            value={values.kubeApiQps}
            range={[5, 1000, 5]}
            onChange={(kubeApiQps) => onChange({ kubeApiQps })}
            error={errors.kubeApiQps}
            hint="Default: 50"
          />
          <LimitField
            id="kube-api-user-qps"
            label="Per user"
            value={values.kubeApiUserQps}
            range={[1, 200, 1]}
            onChange={(kubeApiUserQps) => onChange({ kubeApiUserQps })}
            error={errors.kubeApiUserQps}
            hint="So one user cannot slow down everybody else. Default: 10"
          />
        </FieldGroup>
      </CardContent>
    </Card>
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
    <Field data-invalid={error ? true : undefined}>
      <div className="flex items-center justify-between">
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        <span className="rounded-md bg-muted px-2 py-0.5 font-mono text-sm tabular-nums">{value}</span>
      </div>
      <Slider id={id} value={[value]} min={min} max={max} step={step} onValueChange={(v) => onChange(v[0])} />
      {error ? <FieldError>{error}</FieldError> : <FieldDescription>{hint}</FieldDescription>}
    </Field>
  )
}
