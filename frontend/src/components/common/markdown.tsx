import ReactMarkdown, { type Components } from "react-markdown"
import remarkGfm from "remark-gfm"

// react-markdown renders no raw HTML and drops unsafe link protocols (javascript:), so admin
// supplied text cannot inject scripts. Styles are mapped per element (no typography plugin).
const components: Components = {
  h1: (p) => <h1 className="mt-6 mb-3 text-xl font-semibold tracking-tight first:mt-0" {...p} />,
  h2: (p) => <h2 className="mt-5 mb-2 text-lg font-semibold tracking-tight first:mt-0" {...p} />,
  h3: (p) => <h3 className="mt-4 mb-2 font-semibold first:mt-0" {...p} />,
  h4: (p) => <h4 className="mt-3 mb-1 font-medium first:mt-0" {...p} />,
  p: (p) => <p className="my-2 leading-relaxed" {...p} />,
  ul: (p) => <ul className="my-2 list-disc space-y-1 pl-5" {...p} />,
  ol: (p) => <ol className="my-2 list-decimal space-y-1 pl-5" {...p} />,
  a: (p) => <a className="font-medium underline underline-offset-4" target="_blank" rel="noreferrer" {...p} />,
  hr: (p) => <hr className="my-4 border-border" {...p} />,
  code: (p) => <code className="rounded bg-muted px-1 py-0.5 font-mono text-[0.9em]" {...p} />,
  // Code blocks (license texts): wrapped instead of scrolling sideways; the code inside loses the inline style.
  pre: (p) => (
    <pre
      className="my-3 rounded-md bg-muted p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap [&>code]:bg-transparent [&>code]:p-0"
      {...p}
    />
  ),
  table: (p) => (
    <div className="my-3 overflow-x-auto">
      <table className="w-full border-collapse text-left" {...p} />
    </div>
  ),
  th: (p) => <th className="border-b px-2 py-1 font-semibold" {...p} />,
  td: (p) => <td className="border-b px-2 py-1" {...p} />,
}

/** Renders Markdown: texts written by an administrator (legal texts) and the licenses page. */
export function Markdown({ children }: { children: string }) {
  return (
    <div className="text-sm wrap-break-word text-foreground">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {children}
      </ReactMarkdown>
    </div>
  )
}
