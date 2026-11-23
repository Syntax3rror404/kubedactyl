import { CpuIcon, HardDriveIcon, MemoryStickIcon } from "lucide-react"

import { Slider } from "@/components/ui/slider"
import { formatCpuLimit, formatMiB } from "@/lib/format"

const formatCpuSlider = (v: number) => (v ? formatCpuLimit(v) : "Unlimited (1 core reserved)")

/** Memory, CPU and disk sliders shared by the create wizard and the settings page. */
export function ResourceSliders({
  memory,
  onMemoryChange,
  cpu,
  onCpuChange,
  disk,
  onDiskChange,
  minDisk,
  diskHint,
}: {
  memory: number
  onMemoryChange: (v: number) => void
  cpu: number
  onCpuChange: (v: number) => void
  disk: number
  onDiskChange: (v: number) => void
  minDisk?: number
  diskHint?: string
}) {
  return (
    <div className="grid gap-6">
      <ResourceSlider
        icon={<MemoryStickIcon />}
        label="Memory"
        value={memory}
        onChange={onMemoryChange}
        min={512}
        max={32768}
        step={256}
        format={formatMiB}
      />
      <ResourceSlider
        icon={<CpuIcon />}
        label="CPU"
        value={cpu}
        onChange={onCpuChange}
        min={0}
        max={16000}
        step={250}
        format={formatCpuSlider}
      />
      <ResourceSlider
        icon={<HardDriveIcon />}
        label="Disk"
        value={disk}
        onChange={(v) => onDiskChange(Math.max(v, minDisk ?? 0))}
        min={1024}
        max={204800}
        step={1024}
        format={formatMiB}
        hint={diskHint}
      />
    </div>
  )
}

/** Slider with number input for memory, CPU or disk. */
function ResourceSlider({
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
        <span className="rounded-md bg-muted px-2 py-0.5 text-sm tabular-nums">{format(value)}</span>
      </div>
      <Slider
        value={[value]}
        min={min}
        max={max}
        step={step}
        onValueChange={(v) => onChange(typeof v === "number" ? v : v[0])}
      />
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}
