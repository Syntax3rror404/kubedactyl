import { useState } from "react"
import { FileKeyIcon, KeyRoundIcon, LinkIcon, ShieldCheckIcon, UserCheckIcon } from "lucide-react"

import { CopyButton } from "@/components/common/copy-button"
import { DetailList } from "@/components/common/detail-list"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { CertInfo, ClusterIdentity } from "@/lib/types"

const methodLabel: Record<string, string> = {
  exec: "Exec credential plugin",
  "client-certificate": "Client certificate",
  token: "Bearer token",
  "auth-provider": "Auth provider",
  basic: "Username / password",
  "service-account": "Service account token",
  none: "None",
}

function Cert({ cert }: { cert: CertInfo }) {
  const [now] = useState(Date.now)
  const days = Math.round((new Date(cert.notAfter).getTime() - now) / 86_400_000)
  return (
    <span title={`Issuer: ${cert.issuer}`}>
      {cert.subject} ·{" "}
      <span className={days < 30 ? "text-amber-600 dark:text-amber-400" : ""}>expires in {days} days</span>
    </span>
  )
}

/** How the panel reaches the API server: kubeconfig or in-cluster service account, TLS. */
export function ConnectionCard({ id }: { id: ClusterIdentity }) {
  const kc = id.kubeconfig
  const sa = id.serviceAccount
  const rows: [string, React.ReactNode][] = [
    ["Mode", id.mode === "in-cluster" ? "In-cluster service account" : "Kubeconfig"],
    [
      "API server",
      <span key="server" className="inline-flex items-center gap-1">
        {id.server}
        <CopyButton value={id.server} label="Copy URL" />
      </span>,
    ],
    ["Kubernetes", `${id.version} · ${id.platform}`],
    ["Protocol", id.protocol],
  ]
  if (kc) {
    rows.push(["File", kc.files.join(", ")], ["Context", kc.context], ["Cluster", kc.cluster], ["User entry", kc.user])
    if (kc.namespace) rows.push(["Default namespace", kc.namespace])
  }
  if (sa) {
    rows.push(["Service account", `${sa.namespace}/${sa.name}`])
    if (sa.tokenExpiresAt) rows.push(["Token expires", new Date(sa.tokenExpiresAt).toLocaleString()])
  }
  rows.push([
    "TLS",
    id.tls.insecure ? (
      <span key="tls" className="text-red-500">
        verification disabled
      </span>
    ) : (
      `verified (${id.tls.caSource === "system" ? "system CAs" : id.tls.caSource + " CA"})`
    ),
  ])
  if (id.tls.ca) rows.push(["CA", <Cert key="ca" cert={id.tls.ca} />])
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <LinkIcon className="size-4" />
          Connection
        </CardTitle>
        <CardDescription>How the panel talks to the Kubernetes API.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <DetailList rows={rows} />
        {kc && kc.contexts.length > 1 && (
          <div className="space-y-1.5">
            <div className="text-xs text-muted-foreground">Contexts in this kubeconfig</div>
            <div className="flex flex-wrap gap-1.5">
              {kc.contexts.map((c) => (
                <Badge key={c} variant={c === kc.context ? "default" : "outline"} className="font-mono text-[10px]">
                  {c}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/** Authentication method (secrets are never sent to the browser) and the resolved identity. */
export function AuthCard({ id }: { id: ClusterIdentity }) {
  const a = id.auth
  const rows: [string, React.ReactNode][] = [["Method", methodLabel[a.method] ?? a.method]]
  if (a.execCommand) rows.push(["Command", [a.execCommand, ...(a.execArgs ?? [])].slice(0, 3).join(" ")])
  if (a.execApiVersion) rows.push(["API version", a.execApiVersion])
  if (a.authProvider) rows.push(["Provider", a.authProvider])
  if (a.tokenPreview) rows.push(["Token", a.tokenPreview])
  if (a.clientCertificate) rows.push(["Certificate", <Cert key="cert" cert={a.clientCertificate} />])
  if (a.impersonating) rows.push(["Impersonating", a.impersonating])
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <KeyRoundIcon className="size-4" />
          Authentication
        </CardTitle>
        <CardDescription>Credentials are resolved by the panel; secret values are never shown.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <DetailList rows={rows} />
        {a.execArgs && a.execArgs.length > 2 && (
          <div className="space-y-1 rounded-lg bg-muted/50 p-3 font-mono text-[11px] leading-relaxed break-all text-muted-foreground">
            <FileKeyIcon className="mb-1 size-3.5" />
            {a.execCommand} {a.execArgs.join(" ")}
          </div>
        )}
        <div className="space-y-2 border-t pt-4">
          <div className="flex items-center gap-2 text-sm font-medium">
            <UserCheckIcon className="size-4" />
            Authenticated as
          </div>
          {id.subject ? (
            <>
              <div className="font-mono text-sm">{id.subject.username}</div>
              <div className="flex flex-wrap gap-1.5">
                {id.subject.groups.map((g) => (
                  <Badge key={g} variant="secondary" className="font-mono text-[10px]">
                    {g}
                  </Badge>
                ))}
              </div>
            </>
          ) : (
            <p className="text-xs text-muted-foreground">SelfSubjectReview failed: {id.subjectError}</p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

/** The permissions the panel needs, checked with SelfSubjectAccessReview. */
export function PermissionsCard({ id }: { id: ClusterIdentity }) {
  const missing = id.permissions.filter((p) => !p.allowed).length
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ShieldCheckIcon className="size-4" />
          Permissions
        </CardTitle>
        <CardDescription>
          {missing === 0 ? "Everything the panel needs is allowed." : `${missing} required permission(s) missing.`}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="divide-y rounded-xl border">
          {id.permissions.map((p) => (
            <div key={p.label} className="flex items-center gap-3 px-3 py-2 text-sm">
              <span
                className={`flex size-5 shrink-0 items-center justify-center rounded-full text-[11px] font-bold ${p.allowed ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400" : "bg-red-500/15 text-red-600"}`}
              >
                {p.allowed ? "✓" : "✕"}
              </span>
              <div className="min-w-0 flex-1">
                <div>{p.label}</div>
                <code className="block truncate text-[11px] text-muted-foreground">
                  {p.verb} {p.group ? `${p.group}/` : ""}
                  {p.resource}
                  {p.namespace ? ` -n ${p.namespace}` : ""}
                </code>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
