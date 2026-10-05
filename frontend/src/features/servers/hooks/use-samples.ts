import { useState } from "react"

import { activePhases } from "@/lib/format"
import type { ServerStats } from "@/lib/types"

export type Sample = { t: number; cpu: number; mem: number }

const HISTORY = 60

/** Keeps a rolling window of stats samples for the charts (reset while the server is offline). */
export function useSamples(stats: ServerStats | undefined, updatedAt: number): Sample[] {
  const [state, setState] = useState<{ at: number; samples: Sample[] }>({ at: 0, samples: [] })
  if (stats && updatedAt !== state.at) {
    const active = activePhases.includes(stats.phase)
    const next = active
      ? [
          ...state.samples.slice(-(HISTORY - 1)),
          { t: updatedAt, cpu: stats.cpuMillis ?? 0, mem: stats.memoryBytes ?? 0 },
        ]
      : []
    setState({ at: updatedAt, samples: next })
  }
  return state.samples
}
