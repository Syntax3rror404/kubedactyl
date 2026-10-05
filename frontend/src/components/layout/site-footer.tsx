import { useState, type ReactNode } from "react"
import { Link } from "react-router"

import { Markdown } from "@/components/common/markdown"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { legalDocs, legalTitles, type LegalDoc } from "@/lib/legal"
import { usePanelInfo, useLegalTexts } from "@/lib/queries"
import { cn } from "@/lib/utils"

const repositoryUrl = "https://github.com/Syntax3rror404/kubedactyl"

/**
 * Footer on every page: panel version, the source code and its issue tracker, the imprint and privacy policy when
 * set, and the licenses; children first.
 */
export function SiteFooter({ className, children }: { className?: string; children?: ReactNode }) {
  const legal = useLegalTexts()
  const info = usePanelInfo()
  const [open, setOpen] = useState<LegalDoc | null>(null)
  const docs = legalDocs.filter((d) => legal.data?.[d])
  return (
    <footer
      className={cn(
        "flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground",
        className,
      )}
    >
      {children}
      {info.data && (
        <span className="tabular-nums">
          {info.data.name} v{info.data.version}
        </span>
      )}
      <a href={repositoryUrl} target="_blank" rel="noreferrer" className="hover:text-foreground hover:underline">
        GitHub
      </a>
      <a
        href={`${repositoryUrl}/issues/new`}
        target="_blank"
        rel="noreferrer"
        className="hover:text-foreground hover:underline"
      >
        Report an issue
      </a>
      {docs.map((d) => (
        <button key={d} type="button" className="hover:text-foreground hover:underline" onClick={() => setOpen(d)}>
          {legalTitles[d]}
        </button>
      ))}
      {/* Written by the production build (Go modules and npm packages); not there in dev mode. */}
      {info.data?.mode === "production" && (
        <Link to="/licenses" target="_blank" className="hover:text-foreground hover:underline">
          Licenses
        </Link>
      )}
      <Dialog open={open !== null} onOpenChange={(o) => !o && setOpen(null)}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{open && legalTitles[open]}</DialogTitle>
          </DialogHeader>
          {open && <Markdown>{legal.data?.[open] ?? ""}</Markdown>}
        </DialogContent>
      </Dialog>
    </footer>
  )
}
