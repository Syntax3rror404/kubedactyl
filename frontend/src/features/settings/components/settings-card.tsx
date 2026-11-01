import type { ReactNode } from "react"
import type { LucideIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"

/** A card of the settings page: icon and title, an optional description, then the content. */
export function SettingsCard({
  icon: Icon,
  title,
  description,
  contentClassName,
  children,
}: {
  icon: LucideIcon
  title: string
  description?: ReactNode
  contentClassName?: string
  children: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Icon className="size-4" />
          {title}
        </CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className={contentClassName}>{children}</CardContent>
    </Card>
  )
}

/** A switch with its label; children are the description below the label. */
export function SwitchField({
  id,
  label,
  checked,
  onChange,
  className,
  children,
}: {
  id: string
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
  className?: string
  children?: ReactNode
}) {
  const fieldLabel = <FieldLabel htmlFor={id}>{label}</FieldLabel>
  return (
    <Field orientation="horizontal" className={className}>
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
      {children ? (
        <div>
          {fieldLabel}
          <FieldDescription>{children}</FieldDescription>
        </div>
      ) : (
        fieldLabel
      )}
    </Field>
  )
}

/** Props a settings field gives its input: the id of the label and whether the value is invalid. */
export interface FieldControl {
  id: string
  "aria-invalid": true | undefined
}

/**
 * A field of a settings card: the label (with `aside` on its right), the input from `children`, then the error or
 * else the hint.
 */
export function SettingsField({
  id,
  label,
  aside,
  hint,
  error,
  className,
  children,
}: {
  id: string
  label: string
  aside?: ReactNode
  hint?: ReactNode
  error?: string
  className?: string
  children: (control: FieldControl) => ReactNode
}) {
  const fieldLabel = <FieldLabel htmlFor={id}>{label}</FieldLabel>
  return (
    <Field data-invalid={error ? true : undefined} className={className}>
      {aside ? (
        <div className="flex items-center justify-between gap-2">
          {fieldLabel}
          {aside}
        </div>
      ) : (
        fieldLabel
      )}
      {children({ id, "aria-invalid": error ? true : undefined })}
      {error ? <FieldError>{error}</FieldError> : hint && <FieldDescription>{hint}</FieldDescription>}
    </Field>
  )
}
