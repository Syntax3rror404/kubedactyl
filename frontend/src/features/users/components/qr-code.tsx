import { useMemo } from "react"

import { qrModules } from "@/features/users/lib/qr-code"
import { cn } from "@/lib/utils"

/** QR code as SVG, black on white in both themes, so every camera reads it. */
export function QrCode({ value, className }: { value: string; className?: string }) {
  const data = useMemo(() => qrModules(value), [value])
  const path = data.flatMap((row, y) => row.map((dark, x) => (dark ? `M${x} ${y}h1v1h-1z` : ""))).join("")
  return (
    <svg
      viewBox={`0 0 ${data.length} ${data.length}`}
      shapeRendering="crispEdges"
      role="img"
      aria-label="QR code"
      className={cn("rounded-lg bg-white", className)}
    >
      <path d={path} fill="#000" />
    </svg>
  )
}
