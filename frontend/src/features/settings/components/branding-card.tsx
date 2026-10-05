import { useRef, useState } from "react"
import { ImageIcon, PaletteIcon, Trash2Icon, UploadIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

/** Largest logo or favicon (the server checks the same limit). */
const MAX_IMAGE = 128 * 1024
const IMAGE_TYPES = "image/png,image/jpeg,image/gif,image/webp,image/svg+xml,image/x-icon,image/vnd.microsoft.icon"

export interface BrandingValues {
  brandName: string
  brandTagline: string
  brandLogo: string
  favicon: string
}

/** Name, tagline, logo and favicon of the panel (sidebar, sign-in page, browser tab). */
export function BrandingCard({
  values,
  onChange,
  errors,
}: {
  values: BrandingValues
  onChange: (patch: Partial<BrandingValues>) => void
  errors: Record<string, string>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <PaletteIcon className="size-4" />
          Branding
        </CardTitle>
        <CardDescription>Adding your own logo and icons</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="grid gap-6 md:grid-cols-2">
          <Field data-invalid={!!errors.brandName}>
            <FieldLabel htmlFor="brand-name">Name</FieldLabel>
            <Input
              id="brand-name"
              value={values.brandName}
              maxLength={40}
              onChange={(e) => onChange({ brandName: e.target.value })}
              placeholder="Kubedactyl"
            />
            {errors.brandName && <FieldError>{errors.brandName}</FieldError>}
          </Field>
          <Field data-invalid={!!errors.brandTagline}>
            <FieldLabel htmlFor="brand-tagline">Tagline</FieldLabel>
            <Input
              id="brand-tagline"
              value={values.brandTagline}
              maxLength={80}
              onChange={(e) => onChange({ brandTagline: e.target.value })}
              placeholder="Game servers on Kubernetes"
            />
            {errors.brandTagline && <FieldError>{errors.brandTagline}</FieldError>}
          </Field>
          <ImageField
            id="brand-logo"
            label="Logo"
            hint="Square images look best (shown at 32 and 48 pixels). Empty: the built-in logo."
            value={values.brandLogo}
            onChange={(brandLogo) => onChange({ brandLogo })}
            error={errors.brandLogo}
          />
          <ImageField
            id="favicon"
            label="Favicon"
            hint="The icon of the browser tab (ICO, PNG or SVG). Empty: the built-in icon."
            value={values.favicon}
            onChange={(favicon) => onChange({ favicon })}
            error={errors.favicon}
          />
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

/** An image stored as data URL: preview, choose a file, remove. */
function ImageField({
  id,
  label,
  hint,
  value,
  onChange,
  error,
}: {
  id: string
  label: string
  hint: string
  value: string
  onChange: (dataUrl: string) => void
  error?: string
}) {
  const input = useRef<HTMLInputElement>(null)
  const [tooLarge, setTooLarge] = useState(false)
  const pick = (file?: File) => {
    if (!file) return
    setTooLarge(file.size > MAX_IMAGE)
    if (file.size > MAX_IMAGE) return
    const reader = new FileReader()
    reader.onload = () => onChange(String(reader.result))
    reader.readAsDataURL(file)
  }
  const message = tooLarge ? "The image is larger than 128 KiB." : error
  return (
    <Field data-invalid={!!message}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="flex items-center gap-3">
        <div className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted/40">
          {value ? (
            <img src={value} alt="" className="size-full object-contain" />
          ) : (
            <ImageIcon className="size-5 text-muted-foreground" />
          )}
        </div>
        <input
          ref={input}
          id={id}
          type="file"
          accept={IMAGE_TYPES}
          className="hidden"
          onChange={(e) => {
            pick(e.target.files?.[0])
            e.target.value = ""
          }}
        />
        <Button type="button" variant="outline" size="sm" onClick={() => input.current?.click()}>
          <UploadIcon />
          Choose image
        </Button>
        {value && (
          <Button type="button" variant="ghost" size="sm" onClick={() => onChange("")}>
            <Trash2Icon />
            Remove
          </Button>
        )}
      </div>
      <FieldDescription>{hint}</FieldDescription>
      {message && <FieldError>{message}</FieldError>}
    </Field>
  )
}
