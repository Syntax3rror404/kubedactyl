import { useCallback, useEffect, useRef, useState } from "react"

import { urls } from "@/lib/api"

export interface ConsoleEvent {
  event: "console output" | "install output" | "daemon message" | "status"
  args?: string[]
}

type Listener = (ev: ConsoleEvent) => void

// Reconnects wait 1 s, 2 s, 4 s, ... up to this long (ms).
const MAX_RETRY_MS = 30_000
// Attempts in a row that never opened before the hook gives up (a refused or deleted server, about 3 minutes).
const MAX_FAILED_CONNECTS = 10
// Close code of a connection whose session ended (logout, lost access): reconnecting cannot help.
const POLICY_VIOLATION = 1008

const parse = (data: unknown): ConsoleEvent | null => {
  try {
    return JSON.parse(String(data)) as ConsoleEvent
  } catch {
    return null
  }
}

/**
 * Connects to the console websocket of a server. Output is delivered through
 * `subscribe` (outside of React state, so the terminal can render thousands of
 * lines cheaply). On every (re)connect the server replays its history, which is
 * signalled with a `reset` so the terminal can be cleared first. Reconnects back off and stop (`ended`) when the
 * session ended or the connection keeps being refused.
 */
export function useConsole(server: string) {
  const [connected, setConnected] = useState(false)
  // The server whose connection ended for good (a state per server, so another server connects again).
  const [endedFor, setEndedFor] = useState<string | null>(null)
  const listeners = useRef(new Set<Listener>())
  // Events since the last (re)connect, replayed to listeners that subscribe later.
  const history = useRef<ConsoleEvent[]>([])
  const resetListeners = useRef(new Set<() => void>())
  const socket = useRef<WebSocket | null>(null)

  useEffect(() => {
    let closed = false
    let retry: ReturnType<typeof setTimeout> | undefined
    let attempt = 0
    let failed = 0

    const connect = () => {
      const ws = new WebSocket(urls.console(server))
      let opened = false
      socket.current = ws
      ws.onopen = () => {
        opened = true
        attempt = 0
        failed = 0
        setConnected(true)
        history.current = []
        resetListeners.current.forEach((fn) => fn())
      }
      ws.onmessage = (msg) => {
        const ev = parse(msg.data)
        if (!ev) return
        history.current.push(ev)
        if (history.current.length > 2000) history.current.splice(0, history.current.length - 2000)
        listeners.current.forEach((fn) => fn(ev))
      }
      ws.onclose = (ev) => {
        setConnected(false)
        if (closed) return
        failed = opened ? 0 : failed + 1
        if (ev.code === POLICY_VIOLATION || failed >= MAX_FAILED_CONNECTS) return setEndedFor(server)
        retry = setTimeout(connect, Math.min(1000 * 2 ** attempt++, MAX_RETRY_MS))
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

  return { connected, ended: endedFor === server, subscribe, sendCommand }
}
