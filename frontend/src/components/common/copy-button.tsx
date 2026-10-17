import { useState } from "react"
import { CheckIcon, CopyIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

/** Small icon button that copies a value to the clipboard. */
export function CopyButton({ value, label = "Copy" }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            void navigator.clipboard.writeText(value).then(() => {
              setCopied(true)
              setTimeout(() => setCopied(false), 1500)
            })
          }}
        >
          {copied ? <CheckIcon className="text-emerald-500" /> : <CopyIcon />}
          <span className="sr-only">{label}</span>
        </Button>
      </TooltipTrigger>
      <TooltipContent>{copied ? "Copied" : label}</TooltipContent>
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
