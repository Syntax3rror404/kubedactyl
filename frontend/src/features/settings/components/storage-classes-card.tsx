import { DatabaseIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { FieldError } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { DefaultCell, type Selection, type SelectionHandlers } from "@/features/settings/components/default-cell"
import type { StorageClass } from "@/lib/types"

/** All storage classes of the cluster; the checked ones can be selected for server volumes. */
export function StorageClassesCard({
  classes,
  selection,
  onToggle,
  onDefault,
  fieldError,
}: { classes?: StorageClass[]; selection: Selection; fieldError?: string } & SelectionHandlers) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <DatabaseIcon className="size-4" />
          Storage classes
        </CardTitle>
        <CardDescription>Checked classes can be picked for new servers. A volume keeps its class.</CardDescription>
      </CardHeader>
      <CardContent className="px-0">
        {!classes ? (
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
                        onCheckedChange={(v) => onToggle(sc.name, v === true)}
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
                      <Badge variant={sc.reclaimPolicy === "Retain" ? "secondary" : "outline"}>
                        {sc.reclaimPolicy}
                      </Badge>
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
        {fieldError && <FieldError className="mx-6 mt-3">{fieldError}</FieldError>}
      </CardContent>
    </Card>
  )
}
