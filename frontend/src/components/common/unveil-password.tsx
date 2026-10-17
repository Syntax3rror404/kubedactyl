import { useEffectEvent, useLayoutEffect, useState } from "react"
import type { ComponentProps, ReactNode } from "react"
import { EyeIcon, EyeOffIcon } from "lucide-react"

import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"

const CHARSET = "!§$%&/(){}[]=?*+#<>@0123456789ABCDEFZ7xyz"
// Milliseconds the random characters run, and between two frames.
const DURATION = 900
const INTERVAL = 45

const randomChar = () => CHARSET[Math.floor(Math.random() * CHARSET.length)]

/**
 * Text that decrypts from left to right when `active` turns on: random characters run briefly, then the real text
 * stands. Returns the frame to show, or null while the real text shows (always with reduced motion).
 */
function useUnveil(text: string, active: boolean) {
  const [frame, setFrame] = useState<string | null>(null)
  // The text when the animation starts; typing is blocked while it runs, so it cannot change meanwhile.
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
 * Password input with an eye button (open while the password is readable) that reveals it with a short decrypting
 * animation. A drop-in for `<Input type="password" />` (controlled: `value` + `onChange`).
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

/**
 * A labelled password field: the error (marks the field invalid) or else the hint below it. `invalid` marks it
 * without a message, e.g. while the typed password is too short.
 *
 * Usage:
 *   <PasswordField id="pw-new" label="New password" value={v} onChange={setV} autoComplete="new-password"
 *     error={errors.new} hint="At least 8 characters." />
 */
export function PasswordField({
  id,
  label,
  value,
  onChange,
  autoComplete,
  autoFocus,
  error,
  invalid,
  hint,
}: {
  id: string
  label: ReactNode
  value: string
  onChange: (value: string) => void
  autoComplete: "current-password" | "new-password"
  autoFocus?: boolean
  error?: string
  invalid?: boolean
  hint?: ReactNode
}) {
  const bad = invalid || !!error
  return (
    <Field data-invalid={bad}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <UnveilPassword
        id={id}
        autoComplete={autoComplete}
        autoFocus={autoFocus}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={bad}
      />
      {error ? <FieldError>{error}</FieldError> : hint && <FieldDescription>{hint}</FieldDescription>}
    </Field>
  )
}
