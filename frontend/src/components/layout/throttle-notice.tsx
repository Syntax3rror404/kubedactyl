import { HourglassIcon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { useThrottle } from "@/lib/queries"

/** Floating notice while the panel refuses requests for a moment (Retry-After); pages poll on by themselves. */
export function ThrottleNotice() {
  const throttle = useThrottle()
  if (!throttle) return null
  return (
    <Callout
      tone="warning"
      icon={<HourglassIcon />}
      title={throttle.busy ? "The panel is busy" : "Too many requests"}
      progress
      className="fixed inset-x-4 bottom-4 z-50 mx-auto max-w-md bg-background shadow-lg"
    >
      <p className="text-muted-foreground">
        {throttle.busy
          ? "The Kubernetes API is overloaded. The page continues on its own in a moment."
          : "You sent too many requests. The page continues on its own in a moment."}
      </p>
    </Callout>
  )
}
