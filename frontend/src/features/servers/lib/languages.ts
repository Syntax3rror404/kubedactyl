import { langNames, type LanguageName } from "@uiw/codemirror-extensions-langs"

// File extensions that map to a different language name.
const aliases: Record<string, LanguageName> = { conf: "properties", env: "properties", yml: "yaml" }

/** Picks the syntax highlighting for a file name; the library keys languages by extension. */
export function languageFor(file: string): LanguageName | null {
  const ext = file.split(".").pop()?.toLowerCase() ?? ""
  if (aliases[ext]) return aliases[ext]
  return (langNames as string[]).includes(ext) ? (ext as LanguageName) : null
}
