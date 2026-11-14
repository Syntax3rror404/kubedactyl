/** A list of enabled names plus one default, e.g. storage classes or load balancer pools. */
export interface Selection {
  enabled: string[]
  defaultName: string
}

/** Callbacks of a selection table. */
export interface SelectionHandlers {
  onToggle: (name: string, on: boolean) => void
  onDefault: (name: string) => void
}

/** Enabling adds a name, disabling removes it; the default moves to the first enabled entry when needed. */
export function selectionHandlers(sel: Selection, apply: (sel: Selection) => void): SelectionHandlers {
  return {
    onToggle: (name, on) => {
      const enabled = on ? [...sel.enabled.filter((n) => n !== name), name] : sel.enabled.filter((n) => n !== name)
      apply({ enabled, defaultName: enabled.includes(sel.defaultName) ? sel.defaultName : (enabled[0] ?? "") })
    },
    onDefault: (name) => apply({ ...sel, defaultName: name }),
  }
}
