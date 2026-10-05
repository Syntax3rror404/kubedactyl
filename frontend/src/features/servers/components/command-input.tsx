import { useRef, useState } from "react"
import { ChevronRightIcon, SendHorizonalIcon } from "lucide-react"
import { toast } from "sonner"

import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"

/** Console command line with a history (arrow up / down). */
export function CommandInput({ disabled, onSend }: { disabled: boolean; onSend: (cmd: string) => boolean }) {
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
          placeholder={disabled ? "Server is offline" : "Type a command…"}
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
