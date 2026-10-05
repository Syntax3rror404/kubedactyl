import { useState } from "react"
import { EggIcon } from "lucide-react"
import { Link, useSearchParams } from "react-router"

import { EmptyState, QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { ServerForm } from "@/features/servers/components/server-form"
import { useSharedServerFields } from "@/features/servers/hooks/use-shared-server-fields"
import { useEggs } from "@/lib/queries"

/** /servers/new (admins): create a server from an egg. */
export function ServerNewPage() {
  const eggs = useEggs()
  const [params] = useSearchParams()
  const [picked, setPicked] = useState(params.get("egg") ?? "")
  const shared = useSharedServerFields()

  return (
    <div className="space-y-8">
      <PageHeader
        title="New server"
        description="Choose a template, size it and launch. Installation starts immediately."
      />
      <QueryState
        query={eggs}
        skeleton={<Skeleton className="h-[70vh] rounded-2xl" />}
        empty={
          <EmptyState
            icon={<EggIcon />}
            title="Import an egg first"
            description="Servers are created from eggs (Pterodactyl / Pelican templates)."
          >
            <Button asChild>
              <Link to="/eggs">Go to eggs</Link>
            </Button>
          </EmptyState>
        }
      >
        {(list) => {
          const egg = list.find((e) => e.metadata.name === picked) ?? list[0]
          // Egg specific state (image, variables) starts fresh for every egg.
          return <ServerForm key={egg.metadata.name} egg={egg} eggs={list} onSelectEgg={setPicked} shared={shared} />
        }}
      </QueryState>
    </div>
  )
}
