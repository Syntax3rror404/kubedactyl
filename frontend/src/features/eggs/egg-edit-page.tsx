import { useEffect, useState } from "react"
import { InfoIcon, SaveIcon } from "lucide-react"
import { Link, useNavigate, useParams } from "react-router"
import { toast } from "sonner"

import { Callout } from "@/components/common/callout"
import { EggIcon } from "@/components/common/egg-icon"
import { QueryState } from "@/components/common/query-state"
import { ScrollableTabsList } from "@/components/common/scrollable-tabs-list"
import { PageHeader } from "@/components/layout/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { Tabs, TabsContent, TabsTrigger } from "@/components/ui/tabs"
import { EggSection } from "@/features/eggs/components/editor/egg-section"
import { UnsavedGuard } from "@/features/eggs/components/editor/unsaved-guard"
import { errorCount, firstErrorSection, sections, type SectionId } from "@/features/eggs/lib/egg-draft"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useEgg, useServers, useUpdateEgg } from "@/lib/queries"
import type { Egg } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/** /eggs/:egg/edit (admins): the egg editor as tabs. */
export function EggEditPage() {
  const { egg: name = "" } = useParams()
  const egg = useEgg(name)
  return (
    <QueryState query={egg} skeleton={<Skeleton className="h-[70vh] rounded-2xl" />}>
      {/* A fresh form for every egg (the saved egg replaces the query data without resetting the form). */}
      {(e) => <EggEditor key={e.metadata.uid} egg={e} />}
    </QueryState>
  )
}

function EggEditor({ egg }: { egg: Egg }) {
  const name = egg.metadata.name
  const navigate = useNavigate()
  const servers = useServers().data?.filter((s) => s.spec.eggRef === name).length ?? 0
  const { draft, setDraft, dirty } = useDraft(egg.spec)
  const [tab, setTab] = useState<SectionId>("config")
  const [done, setDone] = useState(false)

  const save = useUpdateEgg(name, {
    onSuccess: () => {
      setDone(true)
      toast.success("Egg saved")
    },
    onError: (err) => {
      const first = firstErrorSection(err)
      if (first) setTab(first)
      failed("save the egg")(err)
    },
  })
  const errors = fieldErrors(save.error)
  // Leave only after the guard saw the saved state.
  useEffect(() => {
    if (done) navigate(`/eggs/${name}`)
  }, [done, name, navigate])

  return (
    <div className="space-y-6">
      <UnsavedGuard when={dirty && !done} />
      <PageHeader
        icon={<EggIcon name={draft.displayName} icon={draft.icon} className="size-14 rounded-xl" />}
        title={`Edit ${egg.spec.displayName}`}
        description="Changes apply at the next start, the install script at the next reinstall."
        actions={
          <>
            <Button variant="ghost" asChild>
              <Link to={`/eggs/${name}`}>Cancel</Link>
            </Button>
            <Button disabled={!dirty || save.isPending} onClick={() => save.mutate(draft)}>
              {save.isPending ? <Spinner /> : <SaveIcon />}
              Save changes
            </Button>
          </>
        }
      />
      {servers > 0 && (
        <Callout tone="info" icon={<InfoIcon />}>
          <p>
            {servers} server{servers === 1 ? " uses" : "s use"} this egg and keep their own image, startup and
            variables.
          </p>
        </Callout>
      )}
      <Tabs value={tab} onValueChange={(v) => setTab(v as SectionId)}>
        <ScrollableTabsList>
          {sections.map((s) => (
            <TabsTrigger key={s.id} value={s.id}>
              {s.title}
              {errorCount(errors, s.id) > 0 && (
                <Badge variant="destructive" className="ml-1 h-4 px-1 text-[10px]">
                  {errorCount(errors, s.id)}
                </Badge>
              )}
            </TabsTrigger>
          ))}
        </ScrollableTabsList>
        {sections.map((s) => (
          <TabsContent key={s.id} value={s.id} className="pt-2">
            <Card>
              <CardHeader>
                <CardTitle>{s.title}</CardTitle>
                <CardDescription>{s.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <EggSection
                  id={s.id}
                  spec={draft}
                  onChange={(patch) => setDraft((d) => ({ ...d, ...patch }))}
                  errors={errors}
                  self={name}
                />
              </CardContent>
            </Card>
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
