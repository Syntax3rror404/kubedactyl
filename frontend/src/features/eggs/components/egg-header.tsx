import type { ReactNode } from "react"

import { EggIcon } from "@/components/common/egg-icon"
import { PageHeader } from "@/components/layout/page-header"
import { Badge } from "@/components/ui/badge"
import type { EggSpec } from "@/lib/types"

/**
 * Header of an egg page (installed or in the library): icon, name, author, features, tags and the description.
 * children are what only that page knows (origin, update state), shown after the author.
 */
export function EggHeader({ spec: s, actions, children }: { spec: EggSpec; actions: ReactNode; children: ReactNode }) {
  return (
    <>
      <PageHeader
        icon={<EggIcon name={s.displayName} icon={s.icon} className="size-14 rounded-xl" />}
        title={s.displayName}
        description={
          <span className="flex flex-wrap items-center gap-2">
            <span>{s.author}</span>
            {children}
            {s.features?.map((f) => (
              <Badge key={f} variant="secondary">
                {f}
              </Badge>
            ))}
            {s.tags?.map((t) => (
              <Badge key={t} variant="outline">
                #{t}
              </Badge>
            ))}
          </span>
        }
        actions={actions}
      />
      {s.description && <p className="max-w-3xl whitespace-pre-line text-muted-foreground">{s.description}</p>}
    </>
  )
}
