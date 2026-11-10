import { AlertTriangleIcon, DatabaseIcon } from "lucide-react"

import { Callout } from "@/components/common/callout"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { FieldError } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { DefaultCell } from "@/features/settings/components/default-cell"
import { SettingsCard } from "@/features/settings/components/settings-card"
import { selectionHandlers, type Selection, type SettingsCardProps } from "@/features/settings/lib/settings-draft"
import { useStorageClasses } from "@/lib/queries"

/** All storage classes of the cluster; the checked ones can be selected for server volumes. */
export function StorageClassesCard({ draft, onChange, errors }: SettingsCardProps) {
  const query = useStorageClasses()
  const classes = query.data
  const selection: Selection = { enabled: draft.storageClasses, defaultName: draft.defaultStorageClass }
  const { onToggle, onDefault } = selectionHandlers(selection, (s) =>
    onChange({ storageClasses: s.enabled, defaultStorageClass: s.defaultName }),
  )
  return (
    <SettingsCard
      icon={DatabaseIcon}
      title="Storage classes"
      description="Checked classes can be picked for new servers. A volume keeps its class."
      contentClassName="px-0"
    >
      {query.error ? (
        <Callout tone="warning" icon={<AlertTriangleIcon />} className="mx-6">
          <p>{query.error.message}</p>
        </Callout>
      ) : !classes ? (
        <Skeleton className="mx-6 h-32" />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10 pl-6" />
              <TableHead>Name</TableHead>
              <TableHead className="hidden md:table-cell">Provisioner</TableHead>
              <TableHead>Reclaim</TableHead>
              <TableHead className="hidden lg:table-cell">Binding</TableHead>
              <TableHead className="hidden sm:table-cell">Expansion</TableHead>
              <TableHead className="w-32 pr-6 text-right" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {classes.map((sc) => {
              const on = selection.enabled.includes(sc.name)
              return (
                <TableRow key={sc.name} data-state={on ? "selected" : undefined}>
                  <TableCell className="pl-6">
                    <Checkbox
                      checked={on}
                      onCheckedChange={(v) => onToggle(sc.name, v)}
                      aria-label={`Enable ${sc.name}`}
                    />
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-sm font-medium">{sc.name}</span>
                    {sc.isDefault && (
                      <Badge variant="outline" className="ml-2 text-[10px]">
                        cluster default
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
                    {sc.provisioner}
                  </TableCell>
                  <TableCell>
                    <Badge variant={sc.reclaimPolicy === "Retain" ? "secondary" : "outline"}>{sc.reclaimPolicy}</Badge>
                  </TableCell>
                  <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
                    {sc.volumeBindingMode}
                  </TableCell>
                  <TableCell className="hidden text-xs sm:table-cell">
                    {sc.allowVolumeExpansion ? "online" : <span className="text-muted-foreground">no</span>}
                  </TableCell>
                  <TableCell className="pr-6 text-right">
                    <DefaultCell name={sc.name} selection={selection} onDefault={onDefault} />
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}
      {errors.storageClasses && <FieldError className="mx-6 mt-3">{errors.storageClasses}</FieldError>}
    </SettingsCard>
  )
}
