import { useRef, useState } from "react"
import { ImageIcon, PaletteIcon, Trash2Icon, UploadIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { FieldGroup } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { SettingsCard, SettingsField } from "@/features/settings/components/settings-card"
import type { SettingsCardProps } from "@/features/settings/lib/settings-draft"

/** Largest logo or favicon (the server checks the same limit). */
const MAX_IMAGE = 128 * 1024
const IMAGE_TYPES = "image/png,image/jpeg,image/gif,image/webp,image/svg+xml,image/x-icon,image/vnd.microsoft.icon"

/** Name, tagline, logo and favicon of the panel (sidebar, sign-in page, browser tab). */
export function BrandingCard({ draft, onChange, errors }: SettingsCardProps) {
  return (
    <SettingsCard icon={PaletteIcon} title="Branding" description="Name, logo and icons of the panel.">
      <FieldGroup className="grid gap-6 md:grid-cols-2">
        <SettingsField id="brand-name" label="Name" error={errors.brandName}>
          {(control) => (
            <Input
              {...control}
              value={draft.brandName}
              maxLength={40}
              onChange={(e) => onChange({ brandName: e.target.value })}
              placeholder="Kubedactyl"
            />
          )}
        </SettingsField>
        <SettingsField id="brand-tagline" label="Tagline" error={errors.brandTagline}>
          {(control) => (
            <Input
              {...control}
              value={draft.brandTagline}
              maxLength={80}
              onChange={(e) => onChange({ brandTagline: e.target.value })}
              placeholder="Game servers on Kubernetes"
            />
          )}
        </SettingsField>
        <ImageField
          id="brand-logo"
          label="Logo"
          hint="Square images look best (shown at 32 and 48 pixels). Empty: the built-in logo."
          value={draft.brandLogo}
          onChange={(brandLogo) => onChange({ brandLogo })}
          error={errors.brandLogo}
        />
        <ImageField
          id="favicon"
          label="Favicon"
          hint="The icon of the browser tab (ICO, PNG or SVG). Empty: the built-in icon."
          value={draft.favicon}
          onChange={(favicon) => onChange({ favicon })}
          error={errors.favicon}
        />
      </FieldGroup>
    </SettingsCard>
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
  return (
    <SettingsField
      id={id}
      label={label}
      hint={hint}
      error={tooLarge ? `The image is larger than ${MAX_IMAGE / 1024} KiB.` : error}
    >
      {(control) => (
        <div className="flex items-center gap-3">
          <div className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted/40">
            {value ? (
              <img src={value} alt="" className="size-full object-contain" />
            ) : (
              <ImageIcon className="size-5 text-muted-foreground" />
            )}
          </div>
          <input
            {...control}
            ref={input}
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
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setTooLarge(false)
                onChange("")
              }}
            >
              <Trash2Icon />
              Remove
            </Button>
          )}
        </div>
      )}
    </SettingsField>
  )
}
