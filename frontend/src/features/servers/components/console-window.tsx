import { CommandInput } from "@/features/servers/components/command-input"
import { Terminal } from "@/features/servers/components/terminal"
import type { useConsole } from "@/features/servers/hooks/use-console"
import { cn } from "@/lib/utils"

/** Terminal window: title bar with connection state, output and command line. */
export function ConsoleWindow({
  title,
  console: c,
  canSend,
}: {
  title: string
  console: ReturnType<typeof useConsole>
  canSend: boolean
}) {
  return (
    <div className="flex min-w-0 flex-col overflow-hidden rounded-2xl border bg-zinc-50 shadow-sm dark:bg-zinc-950">
      <div className="flex items-center gap-2 border-b bg-background/60 px-4 py-2.5 backdrop-blur">
        <div className="flex gap-1.5">
          <span className="size-3 rounded-full bg-red-400/80" />
          <span className="size-3 rounded-full bg-amber-400/80" />
          <span className="size-3 rounded-full bg-emerald-400/80" />
        </div>
        <span className="ml-2 font-mono text-xs text-muted-foreground">{title}</span>
        <span
          className={cn(
            "ml-auto flex items-center gap-1.5 text-xs",
            c.connected ? "text-emerald-600 dark:text-emerald-400" : "text-muted-foreground",
          )}
        >
          <span className={cn("size-1.5 rounded-full", c.connected ? "bg-emerald-500" : "bg-zinc-400")} />
          {c.connected ? "live" : "connecting…"}
        </span>
      </div>
      <div className="h-[60vh] min-h-96 px-3 py-2">
        <Terminal subscribe={c.subscribe} />
      </div>
      <CommandInput disabled={!canSend} onSend={c.sendCommand} />
    </div>
  )
}
