/** Key/value rows of a card (label left, value right in the monospace font). */
export function DetailList({ rows }: { rows: [string, React.ReactNode][] }) {
  return (
    <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
      {rows.map(([label, value]) => (
        <div key={label} className="contents">
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="min-w-0 truncate text-right font-mono text-xs leading-5">{value}</dd>
        </div>
      ))}
    </dl>
  )
}
