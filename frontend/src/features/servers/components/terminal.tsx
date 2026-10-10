import { useEffect, useRef } from "react"
import { FitAddon } from "@xterm/addon-fit"
import { Terminal as XTerm, type ITheme } from "@xterm/xterm"

import "@xterm/xterm/css/xterm.css"

import type { ConsoleEvent } from "@/features/servers/hooks/use-console"
import { useResolvedTheme } from "@/hooks/use-resolved-theme"

const dark: ITheme = {
  background: "#00000000",
  foreground: "#e4e4e7",
  cursor: "#e4e4e7",
  selectionBackground: "#3f3f4680",
  black: "#18181b",
  red: "#f87171",
  green: "#4ade80",
  yellow: "#facc15",
  blue: "#60a5fa",
  magenta: "#c084fc",
  cyan: "#22d3ee",
  white: "#e4e4e7",
  brightBlack: "#71717a",
  brightRed: "#fca5a5",
  brightGreen: "#86efac",
  brightYellow: "#fde047",
  brightBlue: "#93c5fd",
  brightMagenta: "#d8b4fe",
  brightCyan: "#67e8f9",
  brightWhite: "#fafafa",
}

const light: ITheme = {
  background: "#00000000",
  foreground: "#27272a",
  cursor: "#27272a",
  selectionBackground: "#a1a1aa55",
  black: "#27272a",
  red: "#dc2626",
  green: "#15803d",
  yellow: "#a16207",
  blue: "#1d4ed8",
  magenta: "#7e22ce",
  cyan: "#0e7490",
  white: "#52525b",
  brightBlack: "#71717a",
  brightRed: "#ef4444",
  brightGreen: "#16a34a",
  brightYellow: "#b45309",
  brightBlue: "#2563eb",
  brightMagenta: "#9333ea",
  brightCyan: "#0891b2",
  brightWhite: "#18181b",
}

const DAEMON = "\x1b[1;36m[Kubedactyl]\x1b[0m "
const INSTALL = "\x1b[2m"

/** Read-only xterm that renders the console events of a server. */
export function Terminal({
  subscribe,
}: {
  subscribe: (fn: (ev: ConsoleEvent) => void, onReset: () => void) => () => void
}) {
  const el = useRef<HTMLDivElement>(null)
  const term = useRef<XTerm | null>(null)
  const theme = useResolvedTheme()

  useEffect(() => {
    const t = new XTerm({
      convertEol: true,
      disableStdin: true,
      cursorBlink: false,
      cursorStyle: "underline",
      cursorInactiveStyle: "none",
      allowTransparency: true,
      fontFamily: "'JetBrains Mono Variable', ui-monospace, SFMono-Regular, Menlo, monospace",
      fontSize: 13,
      lineHeight: 1.35,
      scrollback: 10000,
      theme: document.documentElement.classList.contains("dark") ? dark : light,
    })
    const fit = new FitAddon()
    t.loadAddon(fit)
    t.open(el.current!)
    // xterm swallows Ctrl/Cmd+C as a terminal key; copy the selection instead.
    t.attachCustomKeyEventHandler((e) => {
      if (e.type !== "keydown" || !(e.ctrlKey || e.metaKey) || e.key.toLowerCase() !== "c") return true
      if (!t.hasSelection()) return true
      if (navigator.clipboard) {
        e.preventDefault()
        navigator.clipboard.writeText(t.getSelection()).catch(() => document.execCommand("copy"))
      }
      // Without the Clipboard API (insecure context) fall through to the native copy event.
      return false
    })
    term.current = t
    const resize = new ResizeObserver(() => {
      try {
        fit.fit()
      } catch {
        // element not visible yet
      }
    })
    resize.observe(el.current!)
    const unsubscribe = subscribe(
      (ev) => {
        const text = ev.args?.[0] ?? ""
        switch (ev.event) {
          case "console output":
            t.writeln(text)
            break
          case "install output":
            t.writeln(INSTALL + text + "\x1b[0m")
            break
          case "daemon message":
            t.writeln(DAEMON + text)
            break
        }
      },
      () => t.reset(),
    )
    return () => {
      unsubscribe()
      resize.disconnect()
      t.dispose()
    }
  }, [subscribe])

  useEffect(() => {
    if (term.current) term.current.options.theme = theme === "dark" ? dark : light
  }, [theme])

  return <div ref={el} className="h-full w-full" />
}
