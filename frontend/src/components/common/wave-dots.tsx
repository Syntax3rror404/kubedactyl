import { useEffect, useRef } from "react"

import { useResolvedTheme } from "@/hooks/use-resolved-theme"
import { cn } from "@/lib/utils"

// Milliseconds between frames: the dots move slowly, 30 frames per second look smooth and halve the work.
const frameTime = 1000 / 30

/** Draws one frame: every dot rises, falls and swells as the waves pass from bottom left to top right (t in ms). */
function drawDots(ctx: CanvasRenderingContext2D, width: number, height: number, t: number, gap: number, lift: number) {
  ctx.clearRect(0, 0, width, height)
  ctx.beginPath()
  for (let y = gap / 2; y < height + gap; y += gap) {
    for (let x = gap / 2; x < width + gap; x += gap) {
      // Distance along the diagonal (y grows downwards): the crests move towards the top right.
      const d = x - y
      // A long wave plus a shorter, slightly turned one, together between -1 and 1.
      const wave = (Math.sin(d * 0.014 - t * 0.0012) + 0.4 * Math.sin((x - 0.6 * y) * 0.03 - t * 0.002)) / 1.4
      const radius = 1 + wave * 0.5
      ctx.moveTo(x + radius, y + wave * lift)
      ctx.arc(x, y + wave * lift, radius, 0, Math.PI * 2)
    }
  }
  ctx.fill()
}

/**
 * A dot grid whose dots rise and fall like waves on water, drawn on a canvas in the element's text color
 * (still for users who prefer reduced motion; the browser pauses it while the tab is hidden).
 *
 * Usage:
 *   <WaveDots className="absolute inset-0 text-border" />
 *   <WaveDots gap={16} lift={3} className="absolute inset-0 text-primary/20" />
 */
export function WaveDots({
  gap = 22,
  lift = 6,
  className,
}: {
  // Distance between the dots in px.
  gap?: number
  // How far a dot moves up and down in px.
  lift?: number
  className?: string
}) {
  const ref = useRef<HTMLCanvasElement>(null)
  // The color comes from CSS and changes with the theme.
  const theme = useResolvedTheme()

  useEffect(() => {
    const canvas = ref.current
    const ctx = canvas?.getContext("2d")
    if (!canvas || !ctx) return
    const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches
    let width = 0
    let height = 0
    let frame = 0
    let last = -Infinity

    const draw = (t: number) => drawDots(ctx, width, height, t, gap, lift)
    const resize = () => {
      const dpr = window.devicePixelRatio || 1
      width = canvas.clientWidth
      height = canvas.clientHeight
      canvas.width = Math.round(width * dpr)
      canvas.height = Math.round(height * dpr)
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
      ctx.fillStyle = getComputedStyle(canvas).color
      draw(still ? 0 : performance.now())
    }
    const tick = (t: number) => {
      frame = requestAnimationFrame(tick)
      if (t - last < frameTime) return
      last = t
      draw(t)
    }

    const observer = new ResizeObserver(resize)
    observer.observe(canvas)
    resize()
    if (!still) frame = requestAnimationFrame(tick)
    return () => {
      observer.disconnect()
      cancelAnimationFrame(frame)
    }
  }, [gap, lift, theme])

  return <canvas ref={ref} aria-hidden="true" className={cn("pointer-events-none", className)} />
}
