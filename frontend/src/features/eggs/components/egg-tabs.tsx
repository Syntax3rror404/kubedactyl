import { CodeEditor } from "@/components/common/code-editor"
import { ScrollableTabsList } from "@/components/common/scrollable-tabs-list"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsTrigger } from "@/components/ui/tabs"
import { EggConfigFiles } from "@/features/eggs/components/egg-config-files"
import { EggOverview } from "@/features/eggs/components/egg-overview"
import { EggVariablesTable } from "@/features/eggs/components/egg-variables-table"
import type { EggSpec } from "@/lib/types"

/** Overview, variables, config files and install script of an egg (installed or in the library). */
export function EggTabs({ spec: s }: { spec: EggSpec }) {
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
            {s.install?.container || "default installer"}
          </Badge>
          <Badge variant="outline" className="font-mono">
            {s.install?.entrypoint || "bash"}
          </Badge>
        </div>
        <CodeEditor value={s.install?.script || "# no install script"} language="sh" readOnly height="65vh" />
      </TabsContent>
    </Tabs>
  )
}
