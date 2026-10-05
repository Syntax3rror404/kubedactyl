/** Replaces {{VAR}} in the startup command for display; the panel sets memory, port and IP itself. */
function previewStartup(startup: string, env: Record<string, string>, memory: number, port?: number) {
  const extra: Record<string, string> = {
    SERVER_MEMORY: String(memory),
    SERVER_PORT: String(port ?? ""),
    SERVER_IP: "0.0.0.0",
  }
  return startup.replace(/{{\s*([A-Z0-9_]+)\s*}}/gi, (m, key: string) => extra[key] ?? env[key] ?? m)
}

/** The startup command as the container runs it, in a terminal-like box. */
export function StartupPreview({
  startup,
  env,
  memory,
  port,
}: {
  startup: string
  env: Record<string, string>
  memory: number
  port?: number
}) {
  return (
    <div className="max-h-40 overflow-auto rounded-lg bg-zinc-950 p-3 font-mono text-xs leading-relaxed break-all text-emerald-300">
      <span className="text-zinc-500">$ </span>
      {previewStartup(startup, env, memory, port)}
    </div>
  )
}
