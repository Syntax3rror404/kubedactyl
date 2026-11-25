import { useState } from "react"
import { Link } from "react-router"
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  ChevronRightIcon,
  ChevronsUpDownIcon,
  InfoIcon,
  LoaderCircleIcon,
  MinusCircleIcon,
  XCircleIcon,
} from "lucide-react"

import { BrandMark } from "@/base/components/brand-mark"
import { readOpen, saveOpen } from "@/base/lib/open-state"
import { NavCornerMark } from "@/base/navigation/nav-badge"
import { type NavCheck, type NavConfig, type NavTone } from "@/base/types"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

const icons = {
  success: [CheckCircle2Icon, "text-success"],
  warning: [AlertTriangleIcon, "text-warning"],
  pending: [LoaderCircleIcon, "text-pending"],
  danger: [XCircleIcon, "text-destructive"],
  info: [InfoIcon, "text-info"],
  muted: [MinusCircleIcon, "text-muted-foreground"],
} as const

const frame: Partial<Record<NavTone, string>> = {
  warning: "border-warning/40",
  danger: "border-destructive/50",
}

// The Base UI panel height, animated while it opens and closes.
const panel =
  "h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-out " +
  "data-ending-style:h-0 data-starting-style:h-0"

const isProblem = (c: NavCheck) => c.tone === "warning" || c.tone === "danger"

/**
 * Top of the sidebar: logo, name and one line of state. With checks it is a short diagnosis (failing checks always
 * listed, a click opens the passing ones and the link to the details); without, it links home.
 */
export function NavHeader({ header }: { header: NavConfig["header"] }) {
  const checks = header.checks ?? []
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        {checks.length > 0 ? (
          <Diagnosis header={header} checks={checks} />
        ) : (
          <SidebarMenuButton size="lg" tooltip={header.name} render={<Link to="/" />}>
            <Brand header={header} />
          </SidebarMenuButton>
        )}
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

function Brand({ header }: { header: NavConfig["header"] }) {
  return (
    <>
      <span className="relative shrink-0">
        <BrandMark className="aspect-square size-8" iconClassName="size-4" />
        {header.tone && <NavCornerMark tone={header.tone} />}
      </span>
      <div className="grid flex-1 text-left text-sm leading-tight">
        <span className="truncate font-semibold">{header.name}</span>
        {header.detail && <span className="truncate text-xs text-muted-foreground">{header.detail}</span>}
      </div>
    </>
  )
}

function Diagnosis({ header, checks }: { header: NavConfig["header"]; checks: NavCheck[] }) {
  const { state, isMobile, setOpen } = useSidebar()
  const [open, setDetails] = useState(() => readOpen("header") ?? false)
  const problems = checks.filter(isProblem)
  const passing = checks.filter((c) => !isProblem(c))
  return (
    <Collapsible
      open={open}
      onOpenChange={(next) => {
        setDetails(next)
        saveOpen("header", next)
      }}
      className={cn(
        "rounded-lg border border-sidebar-border bg-sidebar-accent/40 group-data-[collapsible=icon]:border-0",
        header.tone && frame[header.tone],
      )}
    >
      <CollapsibleTrigger
        render={<SidebarMenuButton size="lg" tooltip={header.name} />}
        // Collapsed to icons the checks are hidden: a click widens the sidebar first.
        onClick={() => state === "collapsed" && !isMobile && setOpen(true)}
      >
        <Brand header={header} />
        <ChevronsUpDownIcon className="ml-auto size-4 text-muted-foreground" />
      </CollapsibleTrigger>
      {problems.length > 0 && <CheckList checks={problems} className="px-2.5 pb-2.5" />}
      <CollapsibleContent className={panel}>
        <div className="space-y-2 border-t px-2.5 py-2 group-data-[collapsible=icon]:hidden">
          {passing.length > 0 && <CheckList checks={passing} />}
          {header.checksLink && (
            <Link
              to={header.checksLink.to}
              className="flex items-center justify-end gap-1 text-xs font-medium hover:underline"
            >
              {header.checksLink.label}
              <ChevronRightIcon className="size-3.5" />
            </Link>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function CheckList({ checks, className }: { checks: NavCheck[]; className?: string }) {
  return (
    <ul className={cn("space-y-1.5 text-xs group-data-[collapsible=icon]:hidden", className)}>
      {checks.map((c) => {
        const [Icon, color] = icons[c.tone]
        return (
          <li key={c.label} className="flex gap-1.5" title={c.message}>
            <Icon className={cn("mt-px size-3.5 shrink-0", color)} />
            <span className="min-w-0">
              <span className="font-medium">{c.label}</span>
              {c.message && <span className="line-clamp-2 text-muted-foreground">{c.message}</span>}
            </span>
          </li>
        )
      })}
    </ul>
  )
}
