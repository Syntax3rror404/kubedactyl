import type { CSSProperties, HTMLAttributes, ReactNode } from "react"

import { cn } from "@/lib/utils"

// Down and tilted left, then up and tilted right; distance and tilt come from the variables the component sets.
const keyframes =
  "@keyframes float-icon { " +
  "0%, 100% { transform: translateY(var(--float-shift)) rotate(calc(var(--float-angle) * -1)) } " +
  "50% { transform: translateY(calc(var(--float-shift) * -1)) rotate(var(--float-angle)) } }"

const float = "animate-[float-icon_var(--float-duration)_ease-in-out_var(--float-delay)_infinite]"
const speedUp = [
  "hover:[animation-duration:calc(var(--float-duration)/2)]",
  "group-hover:[animation-duration:calc(var(--float-duration)/2)]",
]

/**
 * Lets any element float up and down with a slight tilt (icon, emoji, SVG, image, text);
 * still for users who prefer reduced motion. The keyframes come with the component:
 * React puts the `<style>` into the head once, however many icons there are.
 *
 * Usage:
 *   <FloatingIcon><RocketIcon /></FloatingIcon>
 *   <FloatingIcon distance={14} angle={10} duration={1.2}>🚀</FloatingIcon>
 *   <FloatingIcon delay={-0.5}><StarIcon className="size-6" /></FloatingIcon>
 *   <a href="/"><FloatingIcon hoverSpeedUp><HomeIcon /></FloatingIcon> Start</a>
 */
export function FloatingIcon({
  children,
  distance = 6,
  angle = 4,
  duration = 1.6,
  delay = 0,
  hoverSpeedUp = false,
  className,
  style,
  ...props
}: HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode
  // Total way up and down in px (0 = no movement).
  distance?: number
  // Tilt left/right in degrees (0 = no rotation).
  angle?: number
  // Seconds per up and down (smaller = faster).
  duration?: number
  // Delay in seconds, also negative (for staggered icons).
  delay?: number
  // Twice as fast while hovering the icon or a parent with the class "group".
  hoverSpeedUp?: boolean
}) {
  const vars = {
    "--float-shift": `${distance / 2}px`,
    "--float-angle": `${angle}deg`,
    "--float-duration": `${duration}s`,
    "--float-delay": `${delay}s`,
  } as CSSProperties
  return (
    <span
      style={{ ...style, ...vars }}
      className={cn("inline-flex motion-reduce:animate-none", float, hoverSpeedUp && speedUp, className)}
      {...props}
    >
      <style href="floating-icon" precedence="default">
        {keyframes}
      </style>
      {children}
    </span>
  )
}
