import { useRef, useState } from "react"
import { ChevronRightIcon, SendHorizonalIcon } from "lucide-react"
import { toast } from "sonner"

import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"
import { Terminal } from "@/features/servers/components/terminal"
import type { useConsole } from "@/features/servers/hooks/use-console"
import { cn } from "@/lib/utils"

/** Terminal window: title bar with connection state, output and command line (`blocked` says why it is off). */
export function ConsoleWindow({
  title,
  console: c,
  blocked,
}: {
  title: string
  console: ReturnType<typeof useConsole>
  blocked?: string
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
            c.connected ? "text-success" : "text-muted-foreground",
          )}
        >
          <span className={cn("size-1.5 rounded-full", c.connected ? "bg-success" : "bg-zinc-400")} />
          {c.connected ? "live" : c.ended ? "disconnected" : "connecting…"}
        </span>
      </div>
      <div className="h-[60vh] min-h-96 px-3 py-2">
        <Terminal subscribe={c.subscribe} />
      </div>
      <CommandInput blocked={blocked} onSend={c.sendCommand} />
    </div>
  )
}

/** Console command line with a history (arrow up / down); `blocked` turns it off and says why. */
function CommandInput({ blocked, onSend }: { blocked?: string; onSend: (cmd: string) => boolean }) {
  const disabled = blocked !== undefined
  const [value, setValue] = useState("")
  const history = useRef<string[]>([])
  const index = useRef(-1)

  const send = () => {
    const cmd = value.trim()
    if (!cmd) return
    if (!onSend(cmd)) {
      toast.error("Console is not connected")
      return
    }
    history.current = [cmd, ...history.current.filter((h) => h !== cmd)].slice(0, 50)
    index.current = -1
    setValue("")
  }

  return (
    <div className="border-t bg-background/60 p-2 backdrop-blur">
      <InputGroup className="border-0 bg-transparent shadow-none has-[[data-slot=input-group-control]:focus-visible]:ring-0">
        <InputGroupAddon>
          <ChevronRightIcon className="text-emerald-500" />
        </InputGroupAddon>
        <InputGroupInput
          value={value}
          disabled={disabled}
          placeholder={blocked ?? "Type a command…"}
          className="font-mono text-sm"
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") send()
            if (e.key === "ArrowUp" && history.current.length) {
              e.preventDefault()
              index.current = Math.min(index.current + 1, history.current.length - 1)
              setValue(history.current[index.current])
            }
            if (e.key === "ArrowDown") {
              e.preventDefault()
              index.current = Math.max(index.current - 1, -1)
              setValue(index.current < 0 ? "" : history.current[index.current])
            }
          }}
        />
        <InputGroupAddon align="inline-end">
          <InputGroupButton size="icon-xs" disabled={disabled || !value.trim()} onClick={send}>
            <SendHorizonalIcon />
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
    </div>
  )
}
