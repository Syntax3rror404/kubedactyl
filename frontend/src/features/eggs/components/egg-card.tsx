import type { ReactNode } from "react"
import { Link } from "react-router"

import { EggIcon } from "@/components/common/egg-icon"
import { GlowCard } from "@/components/common/glow-card"
import { Badge } from "@/components/ui/badge"
import { EggOrigin } from "@/features/eggs/components/egg-origin"
import { UpdateUrlBadge } from "@/features/eggs/components/update-url-badge"
import { plural } from "@/lib/format"
import type { Egg } from "@/lib/types"

/** Card of an egg on the eggs page. */
export function EggCard({ egg, servers }: { egg: Egg; servers: number }) {
  const s = egg.spec
  return (
    <EggCardLayout
      to={`/eggs/${egg.metadata.name}`}
      name={s.displayName}
      icon={s.icon}
      author={s.author}
      description={s.description}
      tag={<EggOrigin spec={s} className="font-mono text-[10px]" />}
      highlight={servers > 0 ? plural(servers, "server") : undefined}
    >
      <Badge variant="secondary">{plural(s.dockerImages.length, "image")}</Badge>
      <Badge variant="secondary">{plural(s.variables?.length ?? 0, "variable")}</Badge>
      <UpdateUrlBadge spec={s} />
    </EggCardLayout>
  )
}

/**
 * The layout of the egg cards (installed eggs and the library): icon, name, author and a tag below them, the
 * description, and at the bottom the children followed by a green highlight.
 */
export function EggCardLayout({
  to,
  name,
  icon,
  author,
  description,
  tag,
  highlight,
  children,
}: {
  to: string
  name: string
  icon?: string
  author?: string
  description?: string
  tag: ReactNode
  highlight?: string
  children: ReactNode
}) {
  return (
    <Link to={to} className="block h-full rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring">
      <GlowCard>
        <div className="flex h-full flex-col gap-4 p-5">
          <div className="flex items-start gap-3">
            <EggIcon name={name} icon={icon} className="size-12" />
            <div className="min-w-0 flex-1">
              <h3 className="truncate font-semibold tracking-tight">{name}</h3>
              <p className="truncate text-xs text-muted-foreground">{author || "unknown author"}</p>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">{tag}</div>
            </div>
          </div>
          <p className="line-clamp-2 text-sm text-muted-foreground">{description || "No description."}</p>
          <div className="mt-auto flex flex-wrap items-center gap-1.5 text-xs">
            {children}
            {highlight && (
              <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">{highlight}</Badge>
            )}
          </div>
        </div>
      </GlowCard>
    </Link>
  )
}
