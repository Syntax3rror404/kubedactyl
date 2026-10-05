import { useCallback, useEffect, useRef, useState } from "react"

export interface ConsoleEvent {
  event: "console output" | "install output" | "daemon message" | "status"
  args?: string[]
}

type Listener = (ev: ConsoleEvent) => void

/**
 * Connects to the console websocket of a server. Output is delivered through
 * `subscribe` (outside of React state, so the terminal can render thousands of
 * lines cheaply). On every (re)connect the server replays its history, which is
 * signalled with a `reset` so the terminal can be cleared first.
 */
export function useConsole(server: string) {
  const [connected, setConnected] = useState(false)
  const listeners = useRef(new Set<Listener>())
  // Events since the last (re)connect, replayed to listeners that subscribe later.
  const history = useRef<ConsoleEvent[]>([])
  const resetListeners = useRef(new Set<() => void>())
  const socket = useRef<WebSocket | null>(null)

  useEffect(() => {
    let closed = false
    let retry: ReturnType<typeof setTimeout> | undefined
    let attempt = 0

    const connect = () => {
      const proto = window.location.protocol === "https:" ? "wss" : "ws"
      const ws = new WebSocket(`${proto}://${window.location.host}/api/servers/${encodeURIComponent(server)}/ws`)
      socket.current = ws
      ws.onopen = () => {
        attempt = 0
        setConnected(true)
        history.current = []
        resetListeners.current.forEach((fn) => fn())
      }
      ws.onmessage = (msg) => {
        const ev = JSON.parse(msg.data) as ConsoleEvent
        history.current.push(ev)
        if (history.current.length > 2000) history.current.splice(0, history.current.length - 2000)
        listeners.current.forEach((fn) => fn(ev))
      }
      ws.onclose = () => {
        setConnected(false)
        if (closed) return
        attempt++
        retry = setTimeout(connect, Math.min(1000 * attempt, 5000))
      }
    }
    connect()
    return () => {
      closed = true
      clearTimeout(retry)
      socket.current?.close()
    }
  }, [server])

  const subscribe = useCallback((fn: Listener, onReset: () => void) => {
    history.current.forEach(fn)
    listeners.current.add(fn)
    resetListeners.current.add(onReset)
    return () => {
      listeners.current.delete(fn)
      resetListeners.current.delete(onReset)
    }
  }, [])

  const send = useCallback((event: string, arg: string) => {
    const ws = socket.current
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ event, args: [arg] }))
      return true
    }
    return false
  }, [])

  const sendCommand = useCallback((command: string) => send("send command", command), [send])

  return { connected, subscribe, sendCommand }
}
