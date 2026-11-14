import { baseKeys } from "@/base/lib/queries/keys"

/**
 * React Query cache keys: the base ones and this app's. refresh() matches a key exactly, so reloading a list never
 * reloads the details below it (stats, jobs, file listings, diagnostics); refreshAll() reloads a group (keys.allFiles).
 */
export const keys = {
  ...baseKeys,
  cluster: ["cluster"] as const,
  clusterNodes: ["cluster", "nodes"] as const,
  clusterIdentity: ["cluster", "identity"] as const,
  clusterHealth: ["cluster", "health"] as const,
  storageClasses: ["settings", "storage-classes"] as const,
  pools: ["settings", "pools"] as const,
  eggs: ["eggs"] as const,
  egg: (name: string) => ["eggs", name] as const,
  eggExport: (name: string, format: string) => ["eggs", name, "export", format] as const,
  library: ["egg-library"] as const,
  libraryEgg: (repository: string, path: string) => ["egg-library", repository, path] as const,
  libraryRepository: (url: string) => ["egg-library", "repository", url] as const,
  servers: ["servers"] as const,
  server: (server: string) => ["servers", server] as const,
  serverStats: (server: string) => ["servers", server, "stats"] as const,
  serverDiagnostics: (server: string) => ["servers", server, "diagnostics"] as const,
  schedules: (server: string) => ["servers", server, "schedules"] as const,
  backups: (server: string) => ["servers", server, "backups"] as const,
  jobs: (server: string) => ["servers", server, "jobs"] as const,
  /** Group of the listings and contents of a server's files (the session keys are separate); see refreshAll. */
  allFiles: (server: string) => ["servers", server, "files"] as const,
  files: (server: string, dir: string) => ["servers", server, "files", "list", dir] as const,
  fileContent: (server: string, file: string) => ["servers", server, "files", "content", file] as const,
  filesOpen: (server: string) => ["servers", server, "files-open"] as const,
  filesSession: (server: string) => ["servers", server, "files-session"] as const,
  filesPodState: (server: string) => ["servers", server, "files-pod-state"] as const,
}
