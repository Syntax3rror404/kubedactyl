import { GeneralSection } from "@/features/eggs/components/editor/general-section"
import { ImagesSection } from "@/features/eggs/components/editor/images-section"
import { InstallSection } from "@/features/eggs/components/editor/install-section"
import { ProcessSection } from "@/features/eggs/components/editor/process-section"
import { VariablesSection } from "@/features/eggs/components/editor/variables-section"
import type { FieldErrors, SectionId, SpecChange } from "@/features/eggs/lib/egg-draft"
import type { EggSpec } from "@/lib/types"

/** One part of the egg editor (edit page tab or wizard step). */
export function EggSection({
  id,
  spec,
  onChange,
  errors,
  self,
}: {
  id: SectionId
  spec: EggSpec
  onChange: SpecChange
  errors: FieldErrors
  self?: string
}) {
  switch (id) {
    case "config":
      return <GeneralSection spec={spec} onChange={onChange} errors={errors} />
    case "images":
      return <ImagesSection spec={spec} onChange={onChange} errors={errors} />
    case "process":
      return <ProcessSection spec={spec} onChange={onChange} errors={errors} self={self} />
    case "variables":
      return <VariablesSection spec={spec} onChange={onChange} errors={errors} />
    case "install":
      return <InstallSection spec={spec} onChange={onChange} self={self} />
  }
}
