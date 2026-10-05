import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import type { EggSpec } from "@/lib/types"

/** Read-only table of an egg's variables (detail page). */
export function EggVariablesTable({ spec: s }: { spec: EggSpec }) {
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
          {s.variables?.map((v) => (
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
