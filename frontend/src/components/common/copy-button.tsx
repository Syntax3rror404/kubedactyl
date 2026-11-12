import { useEffect, useState } from "react"
import { CheckIcon, CopyIcon, XIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

/** Small icon button that copies a value to the clipboard; shows for a moment whether that worked. */
export function CopyButton({ value, label = "Copy" }: { value: string; label?: string }) {
  const [result, setResult] = useState<"copied" | "failed" | null>(null)
  useEffect(() => {
    if (!result) return
    const id = setTimeout(() => setResult(null), 1500)
    return () => clearTimeout(id)
  }, [result])
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              // The clipboard API exists only on HTTPS pages (and localhost).
              if (!navigator.clipboard) return setResult("failed")
              navigator.clipboard.writeText(value).then(
                () => setResult("copied"),
                () => setResult("failed"),
              )
            }}
          />
        }
      >
        {result === "copied" ? (
          <CheckIcon className="text-success" />
        ) : result === "failed" ? (
          <XIcon className="text-destructive" />
        ) : (
          <CopyIcon />
        )}
        <span className="sr-only">{label}</span>
      </TooltipTrigger>
      <TooltipContent>{result === "copied" ? "Copied" : result === "failed" ? "Could not copy" : label}</TooltipContent>
    </Tooltip>
  )
}

/** A value in a muted box with its copy button (links, tokens, IDs); `wrap` breaks long values instead of cutting. */
export function CopyField({
  value,
  label,
  wrap,
  className,
}: {
  value: string
  label: string
  wrap?: boolean
  className?: string
}) {
  return (
    <div className={cn("flex items-center gap-2 rounded-lg bg-muted px-3 py-1.5", className)}>
      <code className={cn("flex-1 font-mono text-xs", wrap ? "break-all" : "truncate")}>{value}</code>
      <CopyButton value={value} label={label} />
    </div>
  )
}
