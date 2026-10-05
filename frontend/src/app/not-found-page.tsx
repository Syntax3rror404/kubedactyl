import { Link } from "react-router"

import { Button } from "@/components/ui/button"

/** Shown for unknown URLs inside the app. */
export function NotFoundPage() {
  return (
    <div className="flex flex-col items-center gap-4 py-24 text-center">
      <span className="bg-gradient-to-b from-foreground to-foreground/30 bg-clip-text text-7xl font-bold text-transparent">
        404
      </span>
      <p className="text-muted-foreground">This page does not exist.</p>
      <Button asChild variant="outline">
        <Link to="/">Back to the dashboard</Link>
      </Button>
    </div>
  )
}
