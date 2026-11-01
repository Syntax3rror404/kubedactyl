import type { EggSpec } from "@/lib/types"

/** How the panel stops a server of the egg (the rules of the controller's sendStop; signals in any case). */
export function describeStop(stop?: string) {
  if (!stop) return "Graceful container stop (SIGTERM)"
  if (!stop.startsWith("^")) return `Console command “${stop}”`
  const signal = stop.slice(1).toUpperCase()
  if (signal === "C" || signal === "SIGINT") return "Signal SIGINT (^C)"
  if (signal === "SIGTERM" || signal === "SIGABRT") return "Graceful container stop (SIGTERM)"
  return `Signal ${stop} → SIGKILL`
}

/** Image and shell of the install script; empty fields use the panel's defaults. */
export const installRuntime = (s: EggSpec) => ({
  container: s.install?.container || "default installer",
  entrypoint: s.install?.entrypoint || "bash",
})
