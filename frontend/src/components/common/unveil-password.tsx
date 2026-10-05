import { useEffectEvent, useLayoutEffect, useState } from "react"
import type { ComponentProps } from "react"
import { EyeIcon, EyeOffIcon } from "lucide-react"

import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"

const CHARSET = "!§$%&/(){}[]=?*+#<>@0123456789ABCDEFZ7xyz"
// Milliseconds the random characters run, and between two frames.
const DURATION = 900
const INTERVAL = 45

const randomChar = () => CHARSET[Math.floor(Math.random() * CHARSET.length)]

/**
 * Text that decrypts from left to right when `active` turns on: random characters run briefly, then the real
 * text stands. Returns the frame to show, or null when the real text shows (also with reduced motion).
 */
function useUnveil(text: string, active: boolean) {
  const [frame, setFrame] = useState<string | null>(null)
  // The text as it was when the animation started; typing is blocked while it runs.
  const target = useEffectEvent(() => text)

  // Layout effect: the first random frame replaces the text before the browser paints it.
  useLayoutEffect(() => {
    const value = target()
    if (!active || !value || matchMedia("(prefers-reduced-motion: reduce)").matches) return
    const start = performance.now()
    const tick = () => {
      const progress = (performance.now() - start) / DURATION
      if (progress >= 1) return stop()
      const resolved = Math.floor(progress * value.length)
      setFrame([...value].map((c, i) => (i < resolved || c === " " ? c : randomChar())).join(""))
    }
    const id = setInterval(tick, INTERVAL)
    const stop = () => {
      clearInterval(id)
      setFrame(null)
    }
    tick()
    return stop
  }, [active])

  return frame
}

/**
 * Password field with an eye button (open while the password is readable, closed while it shows as dots):
 * reveals the password with a short decrypting animation. A drop-in for `<Input type="password" />`
 * (controlled, `value` + `onChange`).
 */
export function UnveilPassword({ value, readOnly, ...props }: Omit<ComponentProps<"input">, "type">) {
  const [revealed, setRevealed] = useState(false)
  const frame = useUnveil(String(value ?? ""), revealed)

  return (
    <InputGroup>
      <InputGroupInput
        {...props}
        type={revealed ? "text" : "password"}
        value={frame ?? value}
        readOnly={readOnly || frame !== null}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
      />
      <InputGroupAddon align="inline-end">
        <InputGroupButton
          size="icon-xs"
          onClick={() => setRevealed((r) => !r)}
          aria-label={revealed ? "Hide password" : "Show password"}
          aria-pressed={revealed}
        >
          {revealed ? <EyeIcon /> : <EyeOffIcon />}
        </InputGroupButton>
      </InputGroupAddon>
    </InputGroup>
  )
}
