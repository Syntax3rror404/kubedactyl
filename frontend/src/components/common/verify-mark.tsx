import { cn } from "@/lib/utils"

export type VerifyState = "waiting" | "ok" | "fail"

// The keyframes come with the component: React puts the style into the head once.
const keyframes = `
@keyframes verify-spin { to { transform: rotate(1turn) } }
@keyframes verify-draw { from { stroke-dashoffset: 40 } to { stroke-dashoffset: 0 } }
@keyframes verify-pop { 0%, 100% { transform: scale(1) } 50% { transform: scale(1.1) } }
@keyframes verify-shake {
  0%, 100% { transform: translateX(0) } 15% { transform: translateX(-9px) } 30% { transform: translateX(8px) }
  45% { transform: translateX(-6px) } 60% { transform: translateX(4px) } 75% { transform: translateX(-2px) }
}
`

function Keyframes() {
  return (
    <style href="verify-mark" precedence="default">
      {keyframes}
    </style>
  )
}

// Time from the answer until the animation has ended: the pop after a check (0.55 s + 0.4 s) and the shake after a
// cross (0.55 s + 0.55 s).
export const verifyOkMs = 950
export const verifyFailMs = 1100

const radius = 27
const circumference = 2 * Math.PI * radius
// Twelve ticks around the center, each one lighter than the one before, like a classic activity indicator.
const ticks = Array.from({ length: 12 }, (_, i) => i)
const labels: Record<VerifyState, string> = { waiting: "Waiting", ok: "Accepted", fail: "Rejected" }

/** A line that draws itself once the answer is there. */
function Stroke({ d, delay = 400 }: { d: string; delay?: number }) {
  return (
    <path
      d={d}
      strokeWidth={4}
      strokeDasharray={40}
      style={{ animationDelay: `${delay}ms` }}
      className="animate-[verify-draw_0.3s_ease-out_both] motion-reduce:animate-none"
    />
  )
}

/**
 * Waiting for an answer, then the answer: ticks turn while waiting, a ring closes around them when the answer
 * comes and a check draws itself (with a small pop) or a cross does and the mark shakes its head. `label` names
 * the state for screen readers.
 */
export function VerifyMark({ state, label, className }: { state: VerifyState; label?: string; className?: string }) {
  const done = state !== "waiting"
  return (
    <span
      role="status"
      aria-label={label ?? labels[state]}
      className={cn(
        "inline-flex size-20 transition-colors duration-300 motion-reduce:animate-none",
        state === "waiting" && "text-muted-foreground",
        state === "ok" && "animate-[verify-pop_0.4s_0.55s_ease-out_both] text-emerald-500",
        state === "fail" && "animate-[verify-shake_0.55s_0.55s_ease-in-out_both] text-red-500",
        className,
      )}
    >
      <Keyframes />
      <svg viewBox="0 0 64 64" className="size-full" fill="none" stroke="currentColor" strokeLinecap="round">
        <g
          className={cn(
            "origin-center animate-[verify-spin_1s_steps(12)_infinite] transition-opacity duration-300",
            "motion-reduce:animate-none",
            done && "opacity-0",
          )}
        >
          {ticks.map((i) => (
            <line
              key={i}
              x1={32}
              y1={8}
              x2={32}
              y2={17}
              strokeWidth={4}
              opacity={1 - i / 13}
              transform={`rotate(${-i * 30} 32 32)`}
            />
          ))}
        </g>
        <circle
          cx={32}
          cy={32}
          r={radius}
          strokeWidth={3.5}
          strokeDasharray={circumference}
          strokeDashoffset={done ? 0 : circumference}
          transform="rotate(-90 32 32)"
          className="transition-[stroke-dashoffset] duration-500 ease-out motion-reduce:transition-none"
        />
        {state === "ok" && <Stroke d="M20 33 L28 41 L44 24" />}
        {state === "fail" && (
          <>
            <Stroke d="M23 23 L41 41" />
            <Stroke d="M41 23 L23 41" delay={550} />
          </>
        )}
      </svg>
    </span>
  )
}
