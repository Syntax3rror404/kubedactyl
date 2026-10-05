import { Slider } from "@/components/ui/slider"

/** Slider with number input for memory, CPU or disk. */
export function ResourceSlider({
  icon,
  label,
  value,
  onChange,
  min,
  max,
  step,
  format,
  hint,
}: {
  icon: React.ReactNode
  label: string
  value: number
  onChange: (v: number) => void
  min: number
  max: number
  step: number
  format: (v: number) => string
  hint?: string
}) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <span className="flex items-center gap-2 text-sm font-medium [&_svg]:size-4 [&_svg]:text-muted-foreground">
          {icon}
          {label}
        </span>
        <span className="rounded-md bg-muted px-2 py-0.5 font-mono text-sm tabular-nums">{format(value)}</span>
      </div>
      <Slider value={[value]} min={min} max={max} step={step} onValueChange={(v) => onChange(v[0])} />
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}
