import type { EggSpec } from "@/lib/types"
import { fieldErrors } from "@/lib/validation"

/** Validation errors from the API, keyed by field path ("variables.2.envVariable"). */
export type FieldErrors = Record<string, string>

/** Change handler of the editor sections. */
export type SpecChange = (patch: Partial<EggSpec>) => void

export type SectionId = "config" | "images" | "process" | "variables" | "install"

/** The parts of the egg editor: tabs on the edit page, steps in the wizard. */
export const sections: { id: SectionId; title: string; description: string; fields: string[] }[] = [
  {
    id: "config",
    title: "Configuration",
    description: "Name, author and how the egg is listed.",
    fields: ["displayName", "author", "description", "features", "tags", "fileDenylist", "source"],
  },
  {
    id: "images",
    title: "Images & startup",
    description: "Runtime images, the startup command and how the server stops.",
    fields: ["dockerImages", "startup", "stop"],
  },
  {
    id: "process",
    title: "Process management",
    description: "When the server counts as running and which config files are adjusted before every start.",
    fields: ["startupDone", "stripAnsi", "configFiles"],
  },
  {
    id: "variables",
    title: "Variables",
    description: "Settings users see on the Startup page, passed as environment variables.",
    fields: ["variables"],
  },
  {
    id: "install",
    title: "Install script",
    description: "Runs once when a server is installed or reinstalled.",
    fields: ["install"],
  },
]

/** Number of errors that belong to a section. */
export function errorCount(errors: FieldErrors, id: SectionId): number {
  const fields = sections.find((s) => s.id === id)!.fields
  return Object.keys(errors).filter((k) => fields.includes(k.split(".")[0])).length
}

/** The first section with a field error of a failed save; the editor jumps to it. */
export function firstErrorSection(err: Error): SectionId | undefined {
  const errors = fieldErrors(err)
  return sections.find((s) => errorCount(errors, s.id) > 0)?.id
}

/** Starting point of a new egg (Pelican's default installer image and entrypoint). */
export function emptyEggSpec(): EggSpec {
  return {
    displayName: "",
    author: "",
    description: "",
    features: [],
    tags: [],
    dockerImages: [{ name: "", image: "" }],
    startup: "",
    stop: "",
    startupDone: [],
    stripAnsi: false,
    configFiles: [],
    fileDenylist: [],
    install: {
      script: "#!/bin/bash\n# Server files are in /mnt/server\ncd /mnt/server\n",
      container: "ghcr.io/pelican-eggs/installers:debian",
      entrypoint: "bash",
    },
    variables: [],
    source: { updateUrl: "" },
  }
}

/** Copy of an egg for "Duplicate" (like Pelican: new name, no update URL, new UUID on save). */
export function duplicateSpec(spec: EggSpec): EggSpec {
  const copy: EggSpec = structuredClone(spec)
  return { ...copy, displayName: `${spec.displayName} Copy`, source: { updateUrl: "" } }
}

/** "Server Jar File" → "SERVER_JAR_FILE" (like Pelican fills the environment variable). */
export function envFromName(name: string): string {
  return name
    .trim()
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .replace(/[^A-Za-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .toUpperCase()
}

/** Moves an item of a list up or down. */
export function move<T>(list: T[], index: number, by: -1 | 1): T[] {
  const next = [...list]
  const target = index + by
  if (target < 0 || target >= list.length) return list
  ;[next[index], next[target]] = [next[target], next[index]]
  return next
}
