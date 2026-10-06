import { LanguageDescription } from "@codemirror/language"
import { languages } from "@codemirror/language-data"

// Extensions that need another language than the file name gives: config files (key=value), and .cfg, which
// game servers use for their own formats rather than TTCN.
const aliases: Record<string, string | null> = { conf: "properties", env: "properties", cfg: null }

/** Name of the syntax highlighting for a file name (for `CodeEditor`), or null for plain text. */
export function languageFor(file: string): string | null {
  const ext = file.split(".").pop()?.toLowerCase() ?? ""
  if (ext in aliases) return aliases[ext]
  return LanguageDescription.matchFilename(languages, file)?.name ?? null
}
