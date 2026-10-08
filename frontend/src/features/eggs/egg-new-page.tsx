import { useEffect, useState } from "react"
import { ArrowLeftIcon, ArrowRightIcon, CheckIcon, PlusIcon } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router"
import { toast } from "sonner"

import { QueryState } from "@/components/common/query-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Spinner } from "@/components/ui/spinner"
import { EggSection } from "@/features/eggs/components/editor/egg-section"
import { EggSummary } from "@/features/eggs/components/editor/egg-summary"
import { UnsavedGuard } from "@/features/eggs/components/editor/unsaved-guard"
import {
  duplicateSpec,
  emptyEggSpec,
  errorCount,
  firstErrorSection,
  sections,
  type FieldErrors,
} from "@/features/eggs/lib/egg-draft"
import { useDraft } from "@/hooks/use-draft"
import { failed } from "@/lib/notify"
import { useCreateEgg, useEgg } from "@/lib/queries"
import type { EggSpec } from "@/lib/types"
import { cn } from "@/lib/utils"
import { fieldErrors } from "@/lib/validation"

const steps = [...sections, { id: "review" as const, title: "Review", description: "Check the egg and create it." }]

/** New egg, step by step; ?from=<egg> starts with a copy of that egg. */
export function EggNewPage() {
  const [params] = useSearchParams()
  const from = params.get("from") ?? ""
  const source = useEgg(from)
  if (!from) return <EggWizard initial={emptyEggSpec()} />
  return (
    <QueryState query={source} skeleton={<Skeleton className="h-[70vh] rounded-2xl" />}>
      {(egg) => <EggWizard key={from} initial={duplicateSpec(egg.spec)} copyOf={egg.spec.displayName} />}
    </QueryState>
  )
}

/** Required fields of a step, checked before moving on (the API validates everything on create). */
function stepErrors(step: string, s: EggSpec): FieldErrors {
  const e: FieldErrors = {}
  if (step === "config") {
    if (!s.displayName.trim()) e.displayName = "is required"
    if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(s.author ?? "")) e.author = "must be an e-mail address"
  }
  if (step === "images") {
    s.dockerImages.forEach((img, i) => {
      if (!img.image.trim()) e[`dockerImages.${i}.image`] = "is required"
    })
    s.startupCommands?.forEach((c, i) => {
      if (!c.name.trim()) e[`startupCommands.${i}.name`] = "is required"
      if (!c.command.trim()) e[`startupCommands.${i}.command`] = "is required"
    })
  }
  return e
}

function EggWizard({ initial, copyOf }: { initial: EggSpec; copyOf?: string }) {
  const navigate = useNavigate()
  const { draft, setDraft, dirty } = useDraft(initial)
  const [step, setStep] = useState(0)
  const [reached, setReached] = useState(0)
  const [errors, setErrors] = useState<FieldErrors>({})
  const [created, setCreated] = useState("")
  const current = steps[step]

  const create = useCreateEgg({
    onSuccess: (egg) => {
      toast.success(`Egg ${egg.spec.displayName} created`)
      setCreated(egg.metadata.name)
    },
    onError: (err) => {
      setErrors(fieldErrors(err))
      const first = firstErrorSection(err)
      if (first) setStep(sections.findIndex((s) => s.id === first))
      failed("create the egg")(err)
    },
  })
  useEffect(() => {
    if (created) navigate(`/eggs/${created}`)
  }, [created, navigate])

  const go = (to: number) => {
    if (to > step) {
      const e = stepErrors(current.id, draft)
      if (Object.keys(e).length) {
        setErrors(e)
        return
      }
    }
    setErrors({})
    setStep(to)
    setReached((r) => Math.max(r, to))
  }

  return (
    <div className="space-y-6">
      <UnsavedGuard when={dirty && !created} />
      <PageHeader
        title={copyOf ? `Duplicate ${copyOf}` : "New egg"}
        description="A template for game servers. You can change everything later."
      />

      <ol className="grid gap-2 sm:grid-cols-6">
        {steps.map((s, i) => {
          const hasErrors = s.id !== "review" && errorCount(errors, s.id) > 0
          return (
            <li key={s.id}>
              <button
                type="button"
                disabled={i > reached}
                onClick={() => go(i)}
                className={cn(
                  "flex w-full items-center gap-2 rounded-xl border px-3 py-2 text-left text-sm transition-colors disabled:opacity-50",
                  i === step ? "border-primary bg-primary/5" : "hover:bg-muted/50",
                  hasErrors && "border-destructive/60",
                )}
              >
                <span
                  className={cn(
                    "flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold",
                    i < step || (i <= reached && i !== step)
                      ? "bg-emerald-500 text-white"
                      : i === step
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground",
                  )}
                >
                  {i < step ? <CheckIcon className="size-3.5" /> : i + 1}
                </span>
                <span className="truncate">{s.title}</span>
              </button>
            </li>
          )
        })}
      </ol>

      <Card>
        <CardHeader>
          <CardTitle>{current.title}</CardTitle>
          <CardDescription>
            {current.description}
            {current.id === "process" || current.id === "variables" || current.id === "install"
              ? " Optional, you can skip this step."
              : ""}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {current.id === "review" ? (
            <EggSummary spec={draft} />
          ) : (
            <EggSection
              id={current.id}
              spec={draft}
              onChange={(patch) => setDraft((d) => ({ ...d, ...patch }))}
              errors={errors}
            />
          )}
        </CardContent>
      </Card>

      <div className="flex justify-between">
        <Button variant="outline" disabled={step === 0} onClick={() => go(step - 1)}>
          <ArrowLeftIcon />
          Back
        </Button>
        {current.id === "review" ? (
          <Button disabled={create.isPending} onClick={() => create.mutate(draft)}>
            {create.isPending ? <Spinner /> : <PlusIcon />}
            Create egg
          </Button>
        ) : (
          <Button onClick={() => go(step + 1)}>
            Next
            <ArrowRightIcon />
          </Button>
        )}
      </div>
    </div>
  )
}
