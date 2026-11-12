import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { cn } from "@/lib/utils"

const tones = {
  info: "border-info/30 bg-info/5 *:[svg]:text-info",
  warning: "border-warning/30 bg-warning/5 *:[svg]:text-warning",
  danger: "border-destructive/30 bg-destructive/5 *:[svg]:text-destructive",
  success: "border-success/30 bg-success/5 *:[svg]:text-success",
}

// The action gets its own column next to the text (below it on phones) instead of the corner of the library alert.
const actionColumn = [
  "has-data-[slot=alert-action]:pr-2.5",
  "sm:grid-cols-[1fr_auto] sm:has-[>svg]:grid-cols-[auto_1fr_auto]",
].join(" ")
const actionCell = [
  "static mt-2 group-has-[>svg]/alert:col-start-2",
  "sm:col-start-auto! sm:-col-end-1 sm:row-start-1 sm:mt-0 sm:self-center",
].join(" ")

/**
 * The shadcn alert in a color: icon (an svg element), optional bold title, text and an action on the right (below
 * on phones).
 */
export function Callout({
  tone,
  icon,
  title,
  children,
  action,
  progress,
  className,
}: {
  tone: keyof typeof tones
  icon?: React.ReactNode
  title?: React.ReactNode
  children?: React.ReactNode
  action?: React.ReactNode
  // Shows the animated stripe at the top while something is in progress.
  progress?: boolean
  className?: string
}) {
  return (
    <Alert
      className={cn(
        "overflow-hidden",
        action && actionColumn,
        // The library alert keeps two rows for title and text; a single one is centered with icon and action.
        !(title && children) && "items-center *:[svg]:row-span-1",
        tones[tone],
        className,
      )}
    >
      {progress && <ProgressStripe />}
      {icon}
      {title && <AlertTitle>{title}</AlertTitle>}
      {children && (
        <AlertDescription className="group-has-[>svg]/alert:col-start-2 [&_p:not(:last-child)]:mb-1">
          {children}
        </AlertDescription>
      )}
      {action && <AlertAction className={cn(actionCell, title && children && "sm:row-span-2")}>{action}</AlertAction>}
    </Alert>
  )
}

/** Animated stripe along the top edge of a box (relative, overflow-hidden) while something runs. */
export function ProgressStripe() {
  return (
    <div className="absolute inset-x-0 top-0 h-0.5 animate-pulse bg-linear-to-r from-transparent via-info to-transparent" />
  )
}
