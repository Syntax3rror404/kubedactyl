import { Badge } from "@/components/ui/badge"
import type { EggSpec } from "@/lib/types"

/** Overview of an egg before it is created (wizard review step). */
export function EggSummary({ spec }: { spec: EggSpec }) {
  const script = spec.install?.script?.trim() ?? ""
  const rows: { label: string; value: React.ReactNode }[] = [
    { label: "Name", value: spec.displayName || "-" },
    { label: "Author", value: spec.author || "-" },
    {
      label: "Description",
      value: spec.description ? <span className="whitespace-pre-line">{spec.description}</span> : "-",
    },
    {
      label: "Docker images",
      value: (
        <span className="space-y-1">
          {spec.dockerImages.map((img, i) => (
            <span key={i} className="block font-mono text-xs">
              {img.name || img.image} → {img.image}
              {i === 0 && (
                <Badge variant="secondary" className="ml-2">
                  default
                </Badge>
              )}
            </span>
          ))}
        </span>
      ),
    },
    { label: "Startup", value: <code className="text-xs break-all">{spec.startup}</code> },
    { label: "Stop", value: spec.stop ? <code className="text-xs">{spec.stop}</code> : "graceful container stop" },
    {
      label: "Running when",
      value: spec.startupDone?.length
        ? spec.startupDone.map((d) => (
            <code key={d} className="mr-2 text-xs">
              {d}
            </code>
          ))
        : "the container runs",
    },
    {
      label: "Config files",
      value: spec.configFiles?.length
        ? spec.configFiles.map((f) => `${f.file} (${f.replace?.length ?? 0})`).join(", ")
        : "none",
    },
    {
      label: "Variables",
      value: spec.variables?.length ? spec.variables.map((v) => v.envVariable).join(", ") : "none",
    },
    {
      label: "Install",
      value: script
        ? `${spec.install?.container || "default installer"} · ${spec.install?.entrypoint || "bash"} · ${script.split("\n").length} lines`
        : "no install script",
    },
  ]
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 text-sm sm:grid-cols-[10rem_1fr]">
      {rows.map((r) => (
        <div key={r.label} className="contents">
          <dt className="text-muted-foreground">{r.label}</dt>
          <dd className="min-w-0">{r.value}</dd>
        </div>
      ))}
    </dl>
  )
}
