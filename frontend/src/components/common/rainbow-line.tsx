import { cn } from "@/lib/utils"

// The gradient is twice as wide as the line and repeats, so moving it by 200% is seamless.
const keyframes =
  "@keyframes rainbow-slide { from { background-position: 0% 50% } to { background-position: 200% 50% } }"

const stripe =
  "animate-[rainbow-slide_4s_linear_infinite] bg-[linear-gradient(90deg,#ff0040,#ff9500,#ffee00,#00e676,#00b0ff,#7c4dff,#ff00c8,#ff0040)] bg-[length:200%_100%] motion-reduce:animate-none"

/**
 * A glowing rainbow line that slides sideways (still for users who prefer reduced motion).
 * The keyframes come with the component: React puts the `<style>` into the head once, however many lines there are.
 *
 * Usage:
 *   <RainbowLine />
 *   <RainbowLine thickness="h-0.5" speed={6} glow="blur-lg" className="my-8" />
 */
export function RainbowLine({
  thickness = "h-1",
  speed = 4,
  glow = "blur-md",
  className,
}: {
  // Tailwind height class: h-px, h-0.5, h-1 …
  thickness?: string
  // Seconds per pass.
  speed?: number
  // Blur of the glow behind the line.
  glow?: "blur-sm" | "blur-md" | "blur-lg" | "blur-xl"
  className?: string
}) {
  const duration = { animationDuration: `${speed}s` }
  return (
    <div role="separator" aria-hidden="true" className={cn("relative w-full", className)}>
      <style href="rainbow-line" precedence="default">
        {keyframes}
      </style>
      {/* Glow layer (blurred, behind) */}
      <div
        style={duration}
        className={cn("absolute inset-x-0 top-0 scale-y-[4] opacity-80", thickness, stripe, glow)}
      />
      {/* Sharp line */}
      <div style={duration} className={cn("relative rounded-full", thickness, stripe)} />
    </div>
  )
}
