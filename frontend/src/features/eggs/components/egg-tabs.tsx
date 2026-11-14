import { CodeEditor } from "@/base/components/code-editor"
import { ScrollableTabsList } from "@/base/components/scrollable-tabs-list"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsContent, TabsTrigger } from "@/components/ui/tabs"
import { EggOverview } from "@/features/eggs/components/egg-overview"
import { installRuntime } from "@/features/eggs/lib/describe"
import type { EggSpec } from "@/lib/types"

/** Overview, variables, config files and install script of an egg (installed or in the library). */
export function EggTabs({ spec: s }: { spec: EggSpec }) {
  const { container, entrypoint } = installRuntime(s)
  return (
    <Tabs defaultValue="overview">
      <ScrollableTabsList>
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="variables">Variables ({s.variables?.length ?? 0})</TabsTrigger>
        <TabsTrigger value="config">Config files ({s.configFiles?.length ?? 0})</TabsTrigger>
        <TabsTrigger value="install">Install script</TabsTrigger>
      </ScrollableTabsList>

      <TabsContent value="overview" className="grid gap-5 pt-4 lg:grid-cols-2">
        <EggOverview spec={s} />
      </TabsContent>

      <TabsContent value="variables" className="pt-4">
        <EggVariablesTable spec={s} />
      </TabsContent>

      <TabsContent value="config" className="grid gap-4 pt-4 lg:grid-cols-2">
        <EggConfigFiles spec={s} />
      </TabsContent>

      <TabsContent value="install" className="space-y-3 pt-4">
        <div className="flex flex-wrap gap-2 text-sm">
          <Badge variant="outline" className="font-mono">
            {container}
          </Badge>
          <Badge variant="outline" className="font-mono">
            {entrypoint}
          </Badge>
        </div>
        <CodeEditor value={s.install?.script || "# no install script"} language="sh" readOnly height="65vh" />
      </TabsContent>
    </Tabs>
  )
}

/** The config file replacements an egg applies before every start. */
function EggConfigFiles({ spec: s }: { spec: EggSpec }) {
  return (
    <>
      {s.configFiles?.length ? (
        s.configFiles.map((f) => (
          <Card key={f.file}>
            <CardHeader>
              <CardTitle className="font-mono text-sm">{f.file}</CardTitle>
              <CardDescription>{f.parser} parser · applied before every start</CardDescription>
            </CardHeader>
            <CardContent className="space-y-1.5">
              {f.replace?.map((r, i) => (
                <div key={i} className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 font-mono text-xs">
                  <span className="truncate">
                    {r.match}
                    {r.ifValue && <span className="text-muted-foreground"> = {r.ifValue}</span>}
                  </span>
                  <span className="text-muted-foreground">→</span>
                  <span className="truncate text-success">{r.replaceWith || '""'}</span>
                </div>
              ))}
            </CardContent>
          </Card>
        ))
      ) : (
        <p className="text-sm text-muted-foreground">This egg does not modify any config files.</p>
      )}
    </>
  )
}

/** Read-only table of an egg's variables (egg tabs). */
function EggVariablesTable({ spec: s }: { spec: EggSpec }) {
  if (!s.variables?.length) return <p className="text-sm text-muted-foreground">This egg has no variables.</p>
  return (
    <Card className="py-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Variable</TableHead>
            <TableHead>Default</TableHead>
            <TableHead>Rules</TableHead>
            <TableHead className="text-right">Access</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {s.variables.map((v) => (
            <TableRow key={v.envVariable}>
              <TableCell className="max-w-md whitespace-normal">
                <div className="font-medium">{v.name}</div>
                <code className="text-xs text-muted-foreground">{v.envVariable}</code>
                {v.description && <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{v.description}</p>}
              </TableCell>
              <TableCell className="font-mono text-xs">{v.defaultValue || "-"}</TableCell>
              <TableCell className="max-w-56 font-mono text-xs break-all whitespace-normal text-muted-foreground">
                {v.rules}
              </TableCell>
              <TableCell className="text-right">
                <div className="flex justify-end gap-1">
                  {v.userViewable && <Badge variant="secondary">view</Badge>}
                  {v.userEditable && <Badge variant="secondary">edit</Badge>}
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Card>
  )
}
