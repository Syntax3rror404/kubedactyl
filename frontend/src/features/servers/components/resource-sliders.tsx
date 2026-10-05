import { CpuIcon, HardDriveIcon, MemoryStickIcon } from "lucide-react"

import { ResourceSlider } from "@/features/servers/components/resource-slider"
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
