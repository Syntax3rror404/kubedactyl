import { useEffect, useMemo, useState } from "react"
import { LanguageDescription, type LanguageSupport } from "@codemirror/language"
import { languages } from "@codemirror/language-data"
import { EditorView } from "@codemirror/view"
import CodeMirror from "@uiw/react-codemirror"

import { useResolvedTheme } from "@/hooks/use-resolved-theme"

// The panel's monospace font (JetBrains Mono) instead of the browser default.
const monoFont = EditorView.theme({ ".cm-scroller": { fontFamily: "var(--font-mono)" } })

/**
 * CodeMirror editor (bundled, no CDN) with syntax highlighting, used for files, scripts and exports. `language` is a
 * name or alias of @codemirror/language-data ("sh", "yaml"); each language is its own chunk, loaded when first used.
 */
export function CodeEditor({
  value,
  onChange,
  language,
  readOnly,
  height = "60vh",
}: {
  value: string
  onChange?: (v: string) => void
  language?: string | null
  readOnly?: boolean
  height?: string
}) {
  const theme = useResolvedTheme()
  const [loaded, setLoaded] = useState<{ language: string; support: LanguageSupport }>()
  useEffect(() => {
    if (!language) return
    let current = true
    void LanguageDescription.matchLanguageName(languages, language, true)
      ?.load()
      .then((support) => current && setLoaded({ language, support }))
    return () => {
      current = false
    }
  }, [language])
  const support = loaded && loaded.language === language ? loaded.support : null
  const extensions = useMemo(() => [EditorView.lineWrapping, monoFont, ...(support ? [support] : [])], [support])
  return (
    <div className="overflow-hidden rounded-lg border text-sm">
      <CodeMirror
        value={value}
        height={height}
        theme={theme}
        readOnly={readOnly}
        editable={!readOnly}
        extensions={extensions}
        onChange={onChange}
        basicSetup={{ foldGutter: true, highlightActiveLine: !readOnly }}
      />
    </div>
  )
}
