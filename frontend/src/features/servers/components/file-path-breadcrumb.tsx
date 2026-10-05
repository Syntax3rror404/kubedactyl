import { Fragment, useState } from "react"
import { HomeIcon } from "lucide-react"

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { draggedNames, isFileDrag } from "@/features/servers/lib/files"
import { cn } from "@/lib/utils"

/** Clickable path segments below the server root (/home/container); parent segments accept dropped entries. */
export function FilePathBreadcrumb({
  dir,
  onNavigate,
  onDropEntries,
}: {
  dir: string
  onNavigate: (dir: string) => void
  onDropEntries?: (names: string[], target: string) => void
}) {
  const segments = dir.split("/").filter(Boolean)
  const [over, setOver] = useState<string | null>(null)
  const drop = (path: string) =>
    onDropEntries && path !== dir
      ? {
          onDragOver: (ev: React.DragEvent) => {
            if (!isFileDrag(ev)) return
            ev.preventDefault()
            ev.dataTransfer.dropEffect = "move"
            setOver(path)
          },
          onDragLeave: () => setOver(null),
          onDrop: (ev: React.DragEvent) => {
            ev.preventDefault()
            setOver(null)
            onDropEntries(draggedNames(ev), path)
          },
        }
      : {}
  const target = (path: string) =>
    cn("rounded px-1", over === path && "bg-sky-500/15 text-foreground ring-1 ring-sky-500")
  return (
    <Breadcrumb>
      <BreadcrumbList className="text-sm">
        <BreadcrumbItem>
          <BreadcrumbLink
            className={cn("flex cursor-pointer items-center gap-1", target("/"))}
            onClick={() => onNavigate("/")}
            {...drop("/")}
          >
            <HomeIcon className="size-3.5" />
            container
          </BreadcrumbLink>
        </BreadcrumbItem>
        {segments.map((seg, i) => {
          const path = "/" + segments.slice(0, i + 1).join("/")
          return (
            <Fragment key={path}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {i === segments.length - 1 ? (
                  <BreadcrumbPage>{seg}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink
                    className={cn("cursor-pointer", target(path))}
                    onClick={() => onNavigate(path)}
                    {...drop(path)}
                  >
                    {seg}
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
