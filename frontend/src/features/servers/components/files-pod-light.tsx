import { useFilesPodState } from "@/lib/queries"
import { cn } from "@/lib/utils"

// Green and yellow in turn while the file container starts.
const keyframes =
  "@keyframes blink-starting { 0% { background-color: var(--color-emerald-500) } " +
  "50%, 100% { background-color: var(--color-amber-400) } }"

const lights = {
  Ready: { className: "bg-emerald-500", label: "running" },
  Starting: { className: "animate-[blink-starting_1s_steps(1,end)_infinite]", label: "starting" },
  Stopping: { className: "bg-red-500", label: "stopping" },
  Stopped: { className: "bg-red-500", label: "stopped" },
} as const

/**
 * Light on the Files tab: file container running (green), stopped (red) or starting (green/yellow).
 * The keyframes come with the component: React puts the `<style>` into the head once.
 */
export function FilesPodLight({ server }: { server: string }) {
  const { data } = useFilesPodState(server)
  if (!data) return null
  const light = lights[data.state] ?? lights.Stopped
  return (
    <span
      role="status"
      title={`File container ${light.label}`}
      aria-label={`File container ${light.label}`}
      className={cn("inline-block size-2 shrink-0 rounded-full", light.className)}
    >
      <style href="files-pod-light" precedence="default">
        {keyframes}
      </style>
    </span>
  )
}
