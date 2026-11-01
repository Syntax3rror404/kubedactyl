import { useEffect, useState } from "react"

import { setThrottledHandler, type Throttle } from "@/lib/api"

/** The last request the panel refused for now (too many requests, panel busy) until it may be sent again. */
export function useThrottle() {
  const [throttle, setThrottle] = useState<(Throttle & { until: number }) | null>(null)
  useEffect(() => {
    // Shown for at least 5 s, so it does not flicker between the polls of a page.
    setThrottledHandler((t) => setThrottle({ ...t, until: Date.now() + Math.max(t.seconds, 5) * 1000 }))
    return () => setThrottledHandler(null)
  }, [])
  useEffect(() => {
    if (!throttle) return
    const id = setTimeout(() => setThrottle(null), throttle.until - Date.now())
    return () => clearTimeout(id)
  }, [throttle])
  return throttle
}
