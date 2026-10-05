import { FileCogIcon, PlusIcon, Trash2Icon, XIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { FieldErrors } from "@/features/eggs/lib/egg-draft"
import type { ConfigFile, ConfigReplace } from "@/lib/types"

const parsers = ["properties", "yaml", "json", "ini", "xml", "file"]
const valueTypes = [
  { value: "string", label: "Text" },
  { value: "number", label: "Number" },
  { value: "boolean", label: "Boolean" },
]

/** Config files the panel adjusts before every start (egg config.files), one card per file. */
export function ConfigFilesEditor({
  files,
  onChange,
  errors,
}: {
  files: ConfigFile[]
  onChange: (files: ConfigFile[]) => void
  errors: FieldErrors
}) {
  const setFile = (i: number, patch: Partial<ConfigFile>) =>
    onChange(files.map((f, j) => (j === i ? { ...f, ...patch } : f)))
  return (
    <div className="space-y-3">
      {files.map((f, i) => {
        const rows = f.replace ?? []
        const setRow = (r: number, patch: Partial<ConfigReplace>) =>
          setFile(i, { replace: rows.map((x, k) => (k === r ? { ...x, ...patch } : x)) })
        const p = `configFiles.${i}.`
        return (
          <div key={i} className="space-y-3 rounded-xl border p-3">
            <div className="flex flex-wrap items-center gap-2">
              <FileCogIcon className="size-4 text-muted-foreground" />
              <Input
                aria-label="File"
                className="h-8 min-w-48 flex-1 font-mono text-xs"
                value={f.file}
                onChange={(e) => setFile(i, { file: e.target.value })}
                placeholder="server.properties"
                aria-invalid={!!errors[p + "file"]}
              />
              <Select value={f.parser} onValueChange={(parser) => setFile(i, { parser })}>
                <SelectTrigger size="sm" className="w-32" aria-label="Parser">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {parsers.map((x) => (
                    <SelectItem key={x} value={x}>
                      {x}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                type="button"
                size="icon-sm"
                variant="ghost"
                className="text-destructive"
                title="Remove file"
                onClick={() => onChange(files.filter((_, j) => j !== i))}
              >
                <Trash2Icon />
              </Button>
            </div>
            {(errors[p + "file"] || errors[p + "parser"]) && (
              <FieldError>{errors[p + "file"] ?? errors[p + "parser"]}</FieldError>
            )}
            {rows.length > 0 && (
              <div className="space-y-2">
                <div className="hidden grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,8rem)_minmax(0,1fr)_auto] gap-2 px-1 text-xs text-muted-foreground md:grid">
                  <span>Key</span>
                  <span>Value</span>
                  <span>Type</span>
                  <span>Only if current value is</span>
                  <span className="w-8" />
                </div>
                {rows.map((r, k) => (
                  <div
                    key={k}
                    className="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,8rem)_minmax(0,1fr)_auto]"
                  >
                    <Input
                      aria-label="Key"
                      className="h-8 font-mono text-xs"
                      value={r.match}
                      onChange={(e) => setRow(k, { match: e.target.value })}
                      placeholder={f.parser === "file" ? "line prefix" : "server-port"}
                      aria-invalid={!!errors[`${p}replace.${k}.match`]}
                    />
                    <Input
                      aria-label="Value"
                      className="h-8 font-mono text-xs"
                      value={r.replaceWith}
                      onChange={(e) => setRow(k, { replaceWith: e.target.value })}
                      placeholder="{{server.build.default.port}}"
                    />
                    <Select
                      value={r.valueType || "string"}
                      onValueChange={(v) => setRow(k, { valueType: v === "string" ? "" : v })}
                    >
                      <SelectTrigger size="sm" className="w-full" aria-label="Type">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {valueTypes.map((t) => (
                          <SelectItem key={t.value} value={t.value}>
                            {t.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      aria-label="Only if"
                      className="h-8 font-mono text-xs"
                      value={r.ifValue ?? ""}
                      onChange={(e) => setRow(k, { ifValue: e.target.value })}
                      placeholder="always"
                    />
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="ghost"
                      title="Remove"
                      onClick={() => setFile(i, { replace: rows.filter((_, x) => x !== k) })}
                    >
                      <XIcon />
                    </Button>
                  </div>
                ))}
              </div>
            )}
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={() => setFile(i, { replace: [...rows, { match: "", replaceWith: "" }] })}
            >
              <PlusIcon />
              Add replacement
            </Button>
          </div>
        )
      })}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() =>
          onChange([...files, { file: "", parser: "properties", replace: [{ match: "", replaceWith: "" }] }])
        }
      >
        <PlusIcon />
        Add config file
      </Button>
    </div>
  )
}
