import { cn } from "@/lib/utils"

const gradients = [
  "from-emerald-500 to-teal-600",
  "from-sky-500 to-indigo-600",
  "from-fuchsia-500 to-purple-600",
  "from-amber-500 to-orange-600",
  "from-rose-500 to-red-600",
  "from-lime-500 to-green-600",
]

function hash(s: string) {
  let h = 0
  for (const c of s) h = (h * 31 + c.charCodeAt(0)) | 0
  return Math.abs(h)
}

/** Shows the egg icon (Pelican eggs ship one) or a colored monogram. */
export function EggIcon({ name, icon, className }: { name: string; icon?: string; className?: string }) {
  if (icon) {
    return (
      <img src={icon} alt="" className={cn("size-10 shrink-0 rounded-lg object-cover ring-1 ring-border", className)} />
    )
  }
  return (
    <div
      className={cn(
        "flex size-10 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br text-sm font-semibold text-white shadow-inner",
        gradients[hash(name) % gradients.length],
        className,
      )}
    >
      {name.slice(0, 2).toUpperCase()}
    </div>
  )
}
