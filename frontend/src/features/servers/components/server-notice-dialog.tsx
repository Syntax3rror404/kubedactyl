import { useState } from "react"
import { MegaphoneIcon } from "lucide-react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

/**
 * The administrator's notice (panel settings). It opens every time a user opens one of
 * their servers; switching between the tabs of the same server does not show it again.
 */
export function ServerNoticeDialog({ server, notice }: { server: string; notice?: string }) {
  const [acknowledged, setAcknowledged] = useState<string | null>(null)
  if (!notice) return null
  return (
    <AlertDialog open={acknowledged !== server} onOpenChange={(open) => !open && setAcknowledged(server)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <MegaphoneIcon className="size-4" />
            Notice from your administrator
          </AlertDialogTitle>
          {/* Plain text: line breaks are kept, nothing is interpreted as HTML. */}
          <AlertDialogDescription className="whitespace-pre-wrap text-foreground">{notice}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogAction onClick={() => setAcknowledged(server)}>Got it</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
