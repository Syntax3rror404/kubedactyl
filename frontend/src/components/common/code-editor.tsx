import { useMemo } from "react"
import { EditorView } from "@codemirror/view"
import { loadLanguage, type LanguageName } from "@uiw/codemirror-extensions-langs"
import CodeMirror from "@uiw/react-codemirror"

import { useResolvedTheme } from "@/hooks/use-resolved-theme"

// The panel's monospace font (JetBrains Mono) instead of the browser default.
const monoFont = EditorView.theme({ ".cm-scroller": { fontFamily: "var(--font-mono)" } })

/** CodeMirror editor (bundled, no CDN) with syntax highlighting, used for files, scripts and exports. */
export function CodeEditor({
  value,
  onChange,
  language,
  readOnly,
  height = "60vh",
}: {
  value: string
  onChange?: (v: string) => void
  language?: LanguageName | null
  readOnly?: boolean
  height?: string
}) {
  const theme = useResolvedTheme()
  const extensions = useMemo(() => {
    const ext = [EditorView.lineWrapping, monoFont]
    const lang = language ? loadLanguage(language) : null
    if (lang) ext.push(lang)
    return ext
  }, [language])
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
