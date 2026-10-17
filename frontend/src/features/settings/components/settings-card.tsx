import type { ReactNode } from "react"
import type { LucideIcon } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
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
