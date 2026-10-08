/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface ChecksCheck {
  /** @example "metrics" */
  id: string;
  /** @example "Metrics server" */
  label: string;
  /** @example "metrics.k8s.io is not available, no CPU and memory usage" */
  message?: string;
  status: "ok" | "warning" | "error" | "skipped";
}

export type ChecksStatus = "ok" | "warning" | "error" | "skipped";

export interface ClusterinfoAuth {
  authProvider?: string;
  clientCertificate?: ClusterinfoCertInfo;
  execApiVersion?: string;
  /** @example ["oidc-login","get-token"] */
  execArgs?: string[];
  /** @example "kubectl" */
  execCommand?: string;
  impersonating?: string;
  /**
   * Method: client-certificate, token, exec, auth-provider, basic, service-account or none.
   * @example "exec"
   */
  method: string;
  /** @example "eyJh…Qw" */
  tokenPreview?: string;
}

export interface ClusterinfoCPUModelCount {
  /** @example 6 */
  count: number;
  /** @example "AMD Ryzen 7 5700U with Radeon Graphics" */
  model: string;
}

export interface ClusterinfoCertInfo {
  /** @example "CN=kubernetes" */
  issuer: string;
  notAfter: string;
  notBefore: string;
  /** @example "CN=kubernetes" */
  subject: string;
}

export interface ClusterinfoHardware {
  /** @example "M5 Pro 1.03" */
  bios?: string;
  board?: string;
  /** @example 8 */
  cpuCores: number;
  /** @example 4374 */
  cpuMaxMHz: number;
  /** @example "AMD Ryzen 7 5700U with Radeon Graphics" */
  cpuModel: string;
  /** @example 1 */
  cpuSockets: number;
  /** @example 16 */
  cpuThreads: number;
  /** @example 66750640128 */
  memoryBytes: number;
  /** @example "M5 Pro" */
  product?: string;
  /** @example "Version 1.0" */
  productVersion?: string;
  /** @example "GMKtec" */
  vendor?: string;
  /** Virtualized is true when the CPU reports the "hypervisor" flag (VM instead of bare metal). */
  virtualized: boolean;
}

export interface ClusterinfoHealth {
  checkedAt: string;
  checks: ChecksCheck[];
  status: "ok" | "warning" | "error";
}

export interface ClusterinfoIdentity {
  auth: ClusterinfoAuth;
  kubeconfig?: ClusterinfoKubeconfig;
  /**
   * Mode is "kubeconfig" or "in-cluster".
   * @example "kubeconfig"
   */
  mode: string;
  permissions: ClusterinfoPermission[];
  /** @example "linux/amd64" */
  platform: string;
  /** @example "HTTP/1.1" */
  protocol: string;
  /** @example "https://k8s.example.com:6443" */
  server: string;
  serviceAccount?: ClusterinfoServiceAccount;
  subject?: ClusterinfoSubject;
  subjectError?: string;
  tls: ClusterinfoTLS;
  /** @example "v1.36.4" */
  version: string;
}

export interface ClusterinfoKubeconfig {
  /** @example "my-cluster" */
  cluster: string;
  /** @example "my-cluster" */
  context: string;
  /** Contexts lists the names of all contexts in the kubeconfig. */
  contexts: string[];
  /** @example ["/home/user/.kube/config"] */
  files: string[];
  namespace?: string;
  /** @example "oidc-user" */
  user: string;
}

export interface ClusterinfoNode {
  /** @example "amd64" */
  architecture: string;
  /** @example "containerd://2.3.5" */
  containerRuntime: string;
  cpuAllocatableMillis: number;
  /** @example 16000 */
  cpuCapacityMillis: number;
  cpuRequestedMillis: number;
  createdAt: string;
  ephemeralStorageBytes: number;
  hardware?: ClusterinfoHardware;
  hardwareError?: string;
  hardwareProbedAt?: string;
  hardwareProbing: boolean;
  /** @example "192.168.1.41" */
  internalIP: string;
  /** @example "6.18.51-talos" */
  kernelVersion: string;
  /** @example "v1.36.4" */
  kubeletVersion: string;
  memoryAllocatableBytes: number;
  memoryCapacityBytes: number;
  memoryRequestedBytes: number;
  /** @example "node-1" */
  name: string;
  /** @example "Talos (v1.14.1)" */
  osImage: string;
  /** @example 110 */
  podsCapacity: number;
  /** @example 42 */
  podsRunning: number;
  /** @example ["MemoryPressure"] */
  pressure: string[];
  ready: boolean;
  /** @example ["control-plane"] */
  roles: string[];
  /** @example "Ready" */
  status: string;
  taints: string[];
  unschedulable: boolean;
  usage?: ClusterinfoUsage;
}

export interface ClusterinfoNodesResponse {
  nodes: ClusterinfoNode[];
  totals: ClusterinfoTotals;
}

export interface ClusterinfoPermission {
  allowed: boolean;
  /** @example "metrics.k8s.io" */
  group?: string;
  /** @example "Exec into pods" */
  label: string;
  namespace?: string;
  /** @example "pods/exec" */
  resource: string;
  /** @example "create" */
  verb: string;
}

export interface ClusterinfoServiceAccount {
  audiences?: string[];
  /** @example "kubedactyl" */
  name: string;
  /** @example "kubedactyl" */
  namespace: string;
  tokenExpiresAt?: string;
}

export interface ClusterinfoSubject {
  extra?: Record<string, string[]>;
  /** @example ["oidc:kube-admins"] */
  groups: string[];
  uid?: string;
  /** @example "oidc:alice" */
  username: string;
}

export interface ClusterinfoTLS {
  ca?: ClusterinfoCertInfo;
  /**
   * CASource is "kubeconfig", "file", "system" (no CA configured) or "service-account".
   * @example "kubeconfig"
   */
  caSource: string;
  insecure: boolean;
  serverName?: string;
}

export interface ClusterinfoTotals {
  cpuCapacityMillis: number;
  cpuModels: ClusterinfoCPUModelCount[];
  cpuRequestedMillis: number;
  cpuUsedMillis: number;
  memoryCapacityBytes: number;
  memoryRequestedBytes: number;
  memoryUsedBytes: number;
  /** MetricsAvailable is false when metrics-server does not answer. */
  metricsAvailable: boolean;
  nodes: number;
  physicalCores: number;
  podsCapacity: number;
  podsRunning: number;
  readyNodes: number;
}

export interface ClusterinfoUsage {
  /** @example 1250 */
  cpuMillis: number;
  /** @example 12884901888 */
  memoryBytes: number;
}

export interface EgglibraryEgg {
  author: string;
  description: string;
  format: string;
  icon?: string;
  name: string;
  /** Path of the file in the repository. */
  path: string;
  /** Repository is the GitHub repository (https://github.com/<owner>/<repo>). */
  repository: string;
  tags: string[];
  /** URL of the raw file: installing imports it from there and keeps it as update URL. */
  url: string;
  uuid: string;
}

export interface EgglibraryRepository {
  eggs: EgglibraryEgg[];
  /** Error tells why the repository could not be read (its eggs are then missing). */
  error?: string;
  fetchedAt: string;
  url: string;
}

export interface FilesEntry {
  isDirectory: boolean;
  isSymlink: boolean;
  mode: string;
  modifiedAt: string;
  name: string;
  size: number;
}

export interface FilesJob {
  error?: string;
  finishedAt?: string;
  id: string;
  kind: "pull" | "backup" | "restore" | "compress" | "decompress";
  label: string;
  /** Progress of a running job, once known. */
  progress?: FilesProgress | null;
  startedAt: string;
  /** State is running, done, failed or cancelled. */
  state: "running" | "done" | "failed" | "cancelled";
}

export interface FilesProgress {
  done: number;
  file?: string;
  startedAt: string;
  total: number;
  unit: "files" | "bytes";
}

export interface HttpapiAcceptInviteRequest {
  /** @example "Alice" */
  displayName?: string;
  /** @example "a-long-password" */
  password: string;
  token: string;
  /**
   * Username is ignored when the invite names one.
   * @example "alice"
   */
  username?: string;
}

export interface HttpapiBackup {
  createdAt: string;
  /** @example "backup-2026-09-29_040000.tar.gz" */
  name: string;
  /**
   * Path is the file manager path (download with /files/download?file=…).
   * @example "/.backups/backup-2026-09-29_040000.tar.gz"
   */
  path: string;
  size: number;
}

export interface HttpapiBackupList {
  items: HttpapiBackup[];
}

export interface HttpapiBranding {
  favicon?: string;
  logo?: string;
  /** @example "Kubedactyl" */
  name: string;
  /** @example "Game servers on Kubernetes" */
  tagline: string;
}

export interface HttpapiClusterInfo {
  /** @example "my-cluster" */
  context: string;
  /** @example "general-pool" */
  loadBalancerPool: string;
  /** @example "kubedactyl" */
  namespace: string;
  /** @example "https://k8s.example.com:6443" */
  server: string;
  /**
   * StorageClass and LoadBalancerPool are the defaults of the panel settings.
   * @example "longhorn"
   */
  storageClass: string;
  /** @example "v1.36.4" */
  version: string;
}

export interface HttpapiCommandRequest {
  /** @example "say Hello" */
  command: string;
}

export interface HttpapiCreateBackupRequest {
  /**
   * Label is appended to the archive name.
   * @example "before-update"
   */
  label?: string;
}

export interface HttpapiCreateFolderRequest {
  /** @example "plugins" */
  name: string;
  /** @example "/" */
  root?: string;
}

export interface HttpapiCreateInviteRequest {
  /** @example "Alice from the Discord server" */
  note?: string;
  /** @example "user" */
  role?: "admin" | "user";
  /** @example "alice" */
  username?: string;
}

export interface HttpapiCreateServerRequest {
  /** IPv6 also asks for an IPv6 address (dual stack clusters). */
  ipv6?: boolean;
  /** @example 2000 */
  cpuMillis?: number;
  /** @example 10240 */
  diskMiB: number;
  /** @example "Survival" */
  displayName: string;
  /** @example "paper" */
  egg: string;
  environment?: Record<string, string>;
  /** ExternalTrafficPolicy of the service (default Local: the server sees the players' IPs). */
  externalTrafficPolicy?: "Local" | "Cluster";
  /** @example "ghcr.io/pelican-eggs/yolks:java_21" */
  image?: string;
  /**
   * LoadBalancerIP optionally requests a fixed IP from the pool (one per IP family, separated by a comma).
   * @example "192.168.1.70"
   */
  loadBalancerIP?: string;
  /**
   * LoadBalancerPool the address comes from (default: the default of the panel settings).
   * @example "general-pool"
   */
  loadBalancerPool?: string;
  /** @example 4096 */
  memoryMiB: number;
  /**
   * Owner is the username the server belongs to (default: the calling admin).
   * @example "alice"
   */
  owner?: string;
  /** @example [25565] */
  ports: number[];
  skipInstall?: boolean;
  /**
   * StartOnCompletion starts the server once the installation is finished.
   * @example true
   */
  startOnCompletion?: boolean;
  startup?: string;
  /** StartupName picks one of the egg's startup commands ("" = its default). */
  startupName?: string;
  /**
   * StorageClass of the data volume (default: the default of the panel settings).
   * @example "longhorn"
   */
  storageClass?: string;
}

export interface HttpapiCreateTokenRequest {
  /**
   * ExpiresInDays is the lifetime: 1 up to the longest API token lifetime of the panel.
   * @example 90
   */
  expiresInDays: number;
  /** @example "ci" */
  name: string;
}

export interface HttpapiCreateUserRequest {
  /** @example "Alice" */
  displayName?: string;
  /** @example "alice@example.com" */
  email?: string;
  /** MustChangePassword makes the user replace the password after signing in. */
  mustChangePassword?: boolean;
  /** @example "a-long-password" */
  password: string;
  /** @example "user" */
  role?: "admin" | "user";
  /** @example "alice" */
  username: string;
}

export interface HttpapiCreatedInvite {
  createdAt: string;
  /** @example "admin" */
  createdBy?: string;
  /** Expired invites stay listed until they are revoked; their link no longer works. */
  expired: boolean;
  expiresAt: string;
  /** @example "1a2b3c4d5e6f" */
  id: string;
  /** @example "Alice from the Discord server" */
  note?: string;
  /** @example "user" */
  role: "admin" | "user";
  /** @example "1a2b3c4d5e6f.…" */
  token: string;
  /**
   * Username is the name the account gets; empty lets the invited person choose it.
   * @example "alice"
   */
  username?: string;
}

export interface HttpapiCreatedToken {
  createdAt: string;
  /** ExpiresAt is the end of the token: its own expiry or the longest API token lifetime of the panel. */
  expiresAt: string;
  /** @example "1a2b3c4d5e6f" */
  id: string;
  /** @example "ci" */
  name: string;
  /** @example "kdt_1a2b3c4d5e6f_…" */
  token: string;
}

export interface HttpapiDecompressFileRequest {
  /** @example "world.zip" */
  file: string;
  /** @example "/" */
  root?: string;
}

export interface HttpapiEggList {
  items: V1Alpha1Egg[];
}

export interface HttpapiErrorResponse {
  /** @example "server not found" */
  error: string;
  /** Fields holds validation errors per field or variable. */
  fields?: Record<string, string>;
}

export interface HttpapiFileList {
  /** @example "/" */
  directory: string;
  items: FilesEntry[];
}

export interface HttpapiFilesRequest {
  /** @example ["world","logs"] */
  files: string[];
  /** @example "/" */
  root?: string;
}

export interface HttpapiFilesSession {
  error?: string;
  /** @example 60 */
  idleTimeoutSeconds: number;
  /** Message explains a Starting state (e.g. "ContainerCreating"). */
  message?: string;
  ready: boolean;
  /**
   * State is Stopped, Starting, Ready or Stopping.
   * @example "Starting"
   */
  state: "Stopped" | "Starting" | "Ready" | "Stopping";
  /**
   * StopsInSeconds is how long a ready container stays without further file operations
   * (reading the session does not count as one).
   * @example 42
   */
  stopsInSeconds?: number;
}

export interface HttpapiGameServerList {
  items: V1Alpha1GameServer[];
}

export interface HttpapiImportURLRequest {
  /** AutoUpdate turns the hourly update from the URL on (an egg imported again keeps it on). */
  autoUpdate?: boolean;
  /** @example "https://raw.githubusercontent.com/pelican-eggs/minecraft/main/java/paper/egg-paper.json" */
  url: string;
}

export interface HttpapiInviteDetails {
  expiresAt: string;
  /**
   * Username is set when the administrator chose the name; the invited person cannot change it.
   * @example "alice"
   */
  username?: string;
}

export interface HttpapiInviteView {
  createdAt: string;
  /** @example "admin" */
  createdBy?: string;
  /** Expired invites stay listed until they are revoked; their link no longer works. */
  expired: boolean;
  expiresAt: string;
  /** @example "1a2b3c4d5e6f" */
  id: string;
  /** @example "Alice from the Discord server" */
  note?: string;
  /** @example "user" */
  role: "admin" | "user";
  /**
   * Username is the name the account gets; empty lets the invited person choose it.
   * @example "alice"
   */
  username?: string;
}

export interface HttpapiJobList {
  items: FilesJob[];
}

export interface HttpapiLegalTexts {
  legalNotice: string;
  privacyPolicy: string;
}

export interface HttpapiLibraryEgg {
  egg: EgglibraryEgg;
  spec: V1Alpha1EggSpec;
}

export interface HttpapiLibraryList {
  repositories: EgglibraryRepository[];
}

export interface HttpapiLoginRequest {
  /**
   * @maxLength 1024
   * @example "secret-password"
   */
  password: string;
  /**
   * The maximum lengths (usernames have at most 32 characters, auth.MaxPasswordLength) keep a request from
   * filling the limiters and the log with huge keys.
   * @maxLength 64
   * @example "admin"
   */
  username: string;
}

export interface HttpapiLoginResponse {
  expiresAt: string;
  token: string;
  user: HttpapiUserView;
}

export interface HttpapiModuleVersion {
  /** @example "Gin" */
  name: string;
  /** @example "v1.12.0" */
  version: string;
}

export interface HttpapiOIDCSignIn {
  enabled: boolean;
  /**
   * Name is shown on the button: "Sign in with <name>".
   * @example "Keycloak"
   */
  name: string;
}

export interface HttpapiPoolList {
  /**
   * IPv6Missing says why servers get no IPv6 address even from a pool with an IPv6 block
   * ("" when they can or it is unknown).
   * @example "No IPv6 service CIDR · IPv6 off in Cilium"
   */
  ipv6Missing?: string;
  /** Error is set when the pools cannot be read (e.g. Cilium LB IPAM is not installed). */
  error?: string;
  items: SettingsPool[];
}

export interface HttpapiPowerRequest {
  /** @example "start" */
  signal: "start" | "stop" | "restart" | "kill";
}

export interface HttpapiPullFileRequest {
  /** @example "/plugins" */
  directory?: string;
  /** Filename defaults to the last path segment of the URL. */
  filename?: string;
  /** @example "https://example.com/plugin.jar" */
  url: string;
}

export interface HttpapiRenameFileRequest {
  /** @example "old.txt" */
  from: string;
  /** @example "/" */
  root?: string;
  /** @example "new.txt" */
  to: string;
}

export interface HttpapiRequestRates {
  panel?: KubeRate | null;
  user: KubeRate;
}

export interface HttpapiScheduleList {
  items: HttpapiScheduleView[];
  /**
   * TimeZone the cron expressions are evaluated in.
   * @example "Europe/Berlin"
   */
  timeZone: string;
}

export interface HttpapiScheduleView {
  /**
   * Cron is a five-field cron expression ("0 4 * * *") or a descriptor like "@daily",
   * evaluated in the panel's time zone. Either Cron or Event is set.
   * +optional
   */
  cron?: string;
  /** Enabled schedules run automatically; disabled ones can still be run by hand. */
  enabled: boolean;
  /**
   * Event runs the schedule when the server was marked as running ("started") or before it
   * is stopped or restarted ("stopping"; the stop waits until the tasks are done).
   * +optional
   * +kubebuilder:validation:Enum=started;stopping
   */
  event?: "started" | "stopping";
  lastResult?: string;
  lastRunAt?: string;
  /**
   * Name identifies the schedule within the server.
   * +kubebuilder:validation:MinLength=1
   * +kubebuilder:validation:MaxLength=50
   */
  name: string;
  nextRunAt?: string;
  /**
   * OnlyWhenOnline skips the run when the server is not running.
   * +optional
   */
  onlyWhenOnline?: boolean;
  running: boolean;
  /**
   * +kubebuilder:validation:MinItems=1
   * +kubebuilder:validation:MaxItems=10
   */
  tasks: V1Alpha1ScheduleTask[];
}

export interface HttpapiServerDiagnostics {
  checkedAt: string;
  checks: ChecksCheck[];
}

export interface HttpapiServerStats {
  /**
   * CPUMillis is the current CPU usage (1000 = one core); null when unknown.
   * @example 350
   */
  cpuMillis?: number | null;
  /**
   * DiskBytes is the space used on the data volume; null when unknown.
   * @example 524288000
   */
  diskBytes?: number | null;
  /**
   * MemoryBytes is the current memory usage; null when unknown.
   * @example 1073741824
   */
  memoryBytes?: number | null;
  /** @example "Running" */
  phase: V1Alpha1Phase;
  /**
   * UptimeSeconds since the game pod was started.
   * @example 3600
   */
  uptimeSeconds: number;
}

export interface HttpapiSettingsView {
  /**
   * AllowPrivateNetworks lets game servers reach private networks (other namespaces,
   * nodes, the Kubernetes API, the LAN). By default every user namespace gets a network
   * policy that only allows the internet, the cluster DNS and the user's own servers.
   * +optional
   */
  allowPrivateNetworks?: boolean;
  /**
   * APITokenMaxDays is the longest lifetime of an API token (default 90). It also limits the
   * tokens created before, counted from their creation.
   * +optional
   * +kubebuilder:validation:Minimum=1
   * +kubebuilder:validation:Maximum=3650
   */
  apiTokenMaxDays?: number;
  /**
   * BrandLogo and Favicon are images as data URLs (PNG, JPEG, GIF, WebP, SVG or ICO, at most
   * 128 KiB); without a logo the built-in one is shown, without a favicon the browser's default.
   * +optional
   * +kubebuilder:validation:MaxLength=180000
   */
  brandLogo?: string;
  /**
   * BrandName and BrandTagline replace "Kubedactyl" and "Game servers on Kubernetes" in the
   * sidebar, on the sign-in page and in the browser title (the footer keeps the software name).
   * +optional
   * +kubebuilder:validation:MaxLength=40
   */
  brandName?: string;
  /**
   * +optional
   * +kubebuilder:validation:MaxLength=80
   */
  brandTagline?: string;
  /**
   * DefaultLoadBalancerPool is preselected for new servers (one of LoadBalancerPools).
   * +optional
   */
  defaultLoadBalancerPool?: string;
  /**
   * DefaultStorageClass is preselected for new servers (one of StorageClasses).
   * +optional
   */
  defaultStorageClass?: string;
  /**
   * DisableAPIDocs turns the API documentation (Swagger UI at /swagger/) off. The API itself
   * keeps working.
   * +optional
   */
  disableApiDocs?: boolean;
  /**
   * EggLibraries are GitHub repositories (https://github.com/<owner>/<repo>) whose eggs the egg
   * library on the eggs page lists. The panel reads them when the library is opened and keeps
   * nothing of them in the cluster.
   * +optional
   * +kubebuilder:validation:MaxItems=20
   */
  eggLibraries?: string[];
  /**
   * ExternalDomain is shown to users as the server address (domain:port) instead of the
   * load balancer IP. Empty shows the IP.
   * +optional
   */
  externalDomain?: string;
  /**
   * +optional
   * +kubebuilder:validation:MaxLength=180000
   */
  favicon?: string;
  /**
   * KubeAPIQPS is how many requests per second the panel sends to the Kubernetes API at most
   * (default 50, bursts of twice that). It applies at once.
   * +optional
   * +kubebuilder:validation:Minimum=5
   * +kubebuilder:validation:Maximum=1000
   */
  kubeApiQps?: number;
  /**
   * KubeAPIUserQPS is how many requests per second one user may send to the panel (default 10,
   * bursts of twice that); each may lead to Kubernetes API calls. More are refused with 429, so
   * one user cannot use up KubeAPIQPS for everybody. It applies at once.
   * +optional
   * +kubebuilder:validation:Minimum=1
   * +kubebuilder:validation:Maximum=200
   */
  kubeApiUserQps?: number;
  /**
   * LegalNotice (imprint) and PrivacyPolicy are Markdown texts linked in the footer of every
   * page, also before sign-in.
   * +optional
   * +kubebuilder:validation:MaxLength=20000
   */
  legalNotice?: string;
  /**
   * LoadBalancerPools are the Cilium LB IPAM pools that can be selected for servers.
   * +optional
   */
  loadBalancerPools?: string[];
  /**
   * OIDC signs users in through an OpenID Connect identity provider (single sign-on). The client
   * secret is kept in a Secret, not here.
   * +optional
   */
  oidc?: V1Alpha1OIDCSettings;
  /** OIDCClientSecretSet tells administrators whether a client secret is stored (always false for users). */
  oidcClientSecretSet: boolean;
  /**
   * +optional
   * +kubebuilder:validation:MaxLength=20000
   */
  privacyPolicy?: string;
  /**
   * ServerNotice is shown to users every time they open one of their servers (plain text).
   * +optional
   * +kubebuilder:validation:MaxLength=2000
   */
  serverNotice?: string;
  /**
   * SessionHours is how long a sign-in lasts (default 12). It applies to every session, so
   * shortening it also ends older sessions.
   * +optional
   * +kubebuilder:validation:Minimum=1
   * +kubebuilder:validation:Maximum=720
   */
  sessionHours?: number;
  /**
   * StorageClasses can be selected for server volumes.
   * +optional
   */
  storageClasses?: string[];
}

export interface HttpapiSetupRequest {
  /** @example "Administrator" */
  displayName?: string;
  email?: string;
  /** @example "a-long-password" */
  password: string;
  /** Token is the one-time setup token from the panel log (or KUBEDACTYL_SETUP_TOKEN). */
  token: string;
  /** @example "admin" */
  username: string;
}

export interface HttpapiSetupStatus {
  required: boolean;
}

export interface HttpapiStartUpgradeRequest {
  /** @example "0.2.4" */
  version: string;
}

export interface HttpapiStorageClassList {
  items: SettingsStorageClass[];
}

export interface HttpapiSuspendRequest {
  suspended: boolean;
}

export interface HttpapiTokenView {
  createdAt: string;
  /** ExpiresAt is the end of the token: its own expiry or the longest API token lifetime of the panel. */
  expiresAt: string;
  /** @example "1a2b3c4d5e6f" */
  id: string;
  /** @example "ci" */
  name: string;
}

export interface HttpapiTransferRequest {
  /** @example "alice" */
  owner: string;
}

export interface HttpapiUpdatePasswordRequest {
  current: string;
  new: string;
  /** RevokeTokens also revokes every API token of the user (after a leak). */
  revokeTokens?: boolean;
}

export interface HttpapiUpdateSchedulesRequest {
  items: V1Alpha1Schedule[];
}

export interface HttpapiUpdateServerRequest {
  /** IPv6 also asks for an IPv6 address (dual stack clusters); admins only. */
  ipv6?: boolean;
  cpuMillis?: number;
  crashRestart?: boolean;
  diskMiB?: number;
  displayName?: string;
  environment?: Record<string, string>;
  /** ExternalTrafficPolicy of the service; admins only. */
  externalTrafficPolicy?: "Local" | "Cluster";
  image?: string;
  /** LoadBalancerIP fixes the address: one IP, or one per IP family separated by a comma. */
  loadBalancerIP?: string;
  /**
   * LoadBalancerPool moves the server to another enabled pool (its address changes).
   * Users may change it; a fixed IP is cleared unless a new one is given.
   */
  loadBalancerPool?: string;
  memoryMiB?: number;
  ports?: number[];
  startup?: string;
  /** StartupName picks one of the egg's startup commands ("" = its default); owners may change it. */
  startupName?: string;
  stopTimeoutSeconds?: number;
}

export interface HttpapiUpdateSettingsRequest {
  /**
   * AllowPrivateNetworks lets game servers reach private networks (other namespaces,
   * nodes, the Kubernetes API, the LAN). By default every user namespace gets a network
   * policy that only allows the internet, the cluster DNS and the user's own servers.
   * +optional
   */
  allowPrivateNetworks?: boolean;
  /**
   * APITokenMaxDays is the longest lifetime of an API token (default 90). It also limits the
   * tokens created before, counted from their creation.
   * +optional
   * +kubebuilder:validation:Minimum=1
   * +kubebuilder:validation:Maximum=3650
   */
  apiTokenMaxDays?: number;
  /**
   * BrandLogo and Favicon are images as data URLs (PNG, JPEG, GIF, WebP, SVG or ICO, at most
   * 128 KiB); without a logo the built-in one is shown, without a favicon the browser's default.
   * +optional
   * +kubebuilder:validation:MaxLength=180000
   */
  brandLogo?: string;
  /**
   * BrandName and BrandTagline replace "Kubedactyl" and "Game servers on Kubernetes" in the
   * sidebar, on the sign-in page and in the browser title (the footer keeps the software name).
   * +optional
   * +kubebuilder:validation:MaxLength=40
   */
  brandName?: string;
  /**
   * +optional
   * +kubebuilder:validation:MaxLength=80
   */
  brandTagline?: string;
  /**
   * DefaultLoadBalancerPool is preselected for new servers (one of LoadBalancerPools).
   * +optional
   */
  defaultLoadBalancerPool?: string;
  /**
   * DefaultStorageClass is preselected for new servers (one of StorageClasses).
   * +optional
   */
  defaultStorageClass?: string;
  /**
   * DisableAPIDocs turns the API documentation (Swagger UI at /swagger/) off. The API itself
   * keeps working.
   * +optional
   */
  disableApiDocs?: boolean;
  /**
   * EggLibraries are GitHub repositories (https://github.com/<owner>/<repo>) whose eggs the egg
   * library on the eggs page lists. The panel reads them when the library is opened and keeps
   * nothing of them in the cluster.
   * +optional
   * +kubebuilder:validation:MaxItems=20
   */
  eggLibraries?: string[];
  /**
   * ExternalDomain is shown to users as the server address (domain:port) instead of the
   * load balancer IP. Empty shows the IP.
   * +optional
   */
  externalDomain?: string;
  /**
   * +optional
   * +kubebuilder:validation:MaxLength=180000
   */
  favicon?: string;
  /**
   * KubeAPIQPS is how many requests per second the panel sends to the Kubernetes API at most
   * (default 50, bursts of twice that). It applies at once.
   * +optional
   * +kubebuilder:validation:Minimum=5
   * +kubebuilder:validation:Maximum=1000
   */
  kubeApiQps?: number;
  /**
   * KubeAPIUserQPS is how many requests per second one user may send to the panel (default 10,
   * bursts of twice that); each may lead to Kubernetes API calls. More are refused with 429, so
   * one user cannot use up KubeAPIQPS for everybody. It applies at once.
   * +optional
   * +kubebuilder:validation:Minimum=1
   * +kubebuilder:validation:Maximum=200
   */
  kubeApiUserQps?: number;
  /**
   * LegalNotice (imprint) and PrivacyPolicy are Markdown texts linked in the footer of every
   * page, also before sign-in.
   * +optional
   * +kubebuilder:validation:MaxLength=20000
   */
  legalNotice?: string;
  /**
   * LoadBalancerPools are the Cilium LB IPAM pools that can be selected for servers.
   * +optional
   */
  loadBalancerPools?: string[];
  /**
   * OIDC signs users in through an OpenID Connect identity provider (single sign-on). The client
   * secret is kept in a Secret, not here.
   * +optional
   */
  oidc?: V1Alpha1OIDCSettings;
  /** OIDCClientSecret replaces the stored client secret (empty removes it); omitted keeps it. */
  oidcClientSecret?: string | null;
  /**
   * +optional
   * +kubebuilder:validation:MaxLength=20000
   */
  privacyPolicy?: string;
  /**
   * ServerNotice is shown to users every time they open one of their servers (plain text).
   * +optional
   * +kubebuilder:validation:MaxLength=2000
   */
  serverNotice?: string;
  /**
   * SessionHours is how long a sign-in lasts (default 12). It applies to every session, so
   * shortening it also ends older sessions.
   * +optional
   * +kubebuilder:validation:Minimum=1
   * +kubebuilder:validation:Maximum=720
   */
  sessionHours?: number;
  /**
   * StorageClasses can be selected for server volumes.
   * +optional
   */
  storageClasses?: string[];
}

export interface HttpapiUpdateUserRequest {
  disabled?: boolean;
  displayName?: string;
  email?: string;
  /** MustChangePassword makes the user replace the password after the next sign-in. */
  mustChangePassword?: boolean;
  password?: string;
  role?: "admin" | "user";
}

export interface HttpapiUpgradeStatus {
  /** @example "oci://ghcr.io/syntax3rror404/charts/kubedactyl" */
  chart?: string;
  checkedAt?: string;
  /** @example "0.2.3" */
  current: string;
  /** Enabled is false without the chart value selfUpgrade.enabled. */
  enabled: boolean;
  error?: string;
  /**
   * IntervalSeconds is how often the registry is checked.
   * @example 600
   */
  intervalSeconds: number;
  jobs: SelfupgradeJob[];
  /** @example "0.2.4" */
  latest?: string;
  /** Newer lists the versions above the current one, highest first. */
  newer: string[];
}

export interface HttpapiUserView {
  createdAt: string;
  disabled: boolean;
  /** @example "Alice" */
  displayName?: string;
  /** @example "alice@example.com" */
  email?: string;
  /** HasPassword is false for accounts that sign in only through the identity provider. */
  hasPassword: boolean;
  lastLoginAt?: string;
  /** MustChangePassword: the user has to replace the password set by an administrator first. */
  mustChangePassword: boolean;
  /** @example "kubedactyl-user-alice" */
  namespace: string;
  /**
   * OIDC is the user of the identity provider the account is linked to (which sets display name, email and
   * role); null for accounts that are not linked.
   */
  oidc?: V1Alpha1OIDCIdentity | null;
  /** @example "user" */
  role: "admin" | "user";
  /** @example 2 */
  servers: number;
  tokens: HttpapiTokenView[];
  /** @example "alice" */
  username: string;
}

export interface HttpapiVersions {
  /** @example "go1.27.1" */
  go: string;
  modules: HttpapiModuleVersion[];
  /** @example "0.2.37" */
  panel: string;
}

export interface HttpserverHealthResponse {
  /** @example "ok" */
  status: string;
}

export interface HttpserverInfoResponse {
  /** @example "production" */
  mode: "development" | "production";
  /** @example "Kubedactyl" */
  name: string;
  /** @example "0.1.0" */
  version: string;
}

export interface KubeRate {
  /** @example 10 */
  limit: number;
  /** @example 2.4 */
  rate: number;
}

export interface SelfupgradeJob {
  finishedAt?: string;
  from: string;
  /** Log holds the last lines of the job output (for failed jobs). */
  log?: string;
  name: string;
  startedAt: string;
  /**
   * State is running, succeeded or failed.
   * @example "running"
   */
  state: "running" | "succeeded" | "failed";
  version: string;
}

export interface SettingsPool {
  /**
   * Blocks are the address ranges ("10.0.0.0/24" or "10.0.0.10-10.0.0.20").
   * @example ["192.168.1.60-192.168.1.120"]
   */
  blocks: string[];
  conflict: boolean;
  disabled: boolean;
  /**
   * Families split the addresses by IP family (IPv4 first), counted by the panel: Cilium counts both
   * together. Only the pools API and the health check fill them in.
   */
  families?: SettingsPoolFamily[];
  ipsAvailable: number;
  /**
   * IPs as reported by Cilium in the pool status (-1 when unknown). Numbers, not integers:
   * an IPv6 block holds more addresses than an int64 (a /64 has 2^64).
   */
  ipsTotal: number;
  ipsUsed: number;
  /** @example "general-pool" */
  name: string;
  reason?: string;
  /** Selectable is false when the pool cannot be targeted by service labels. */
  selectable: boolean;
  /** ServiceLabels are set on a server's service so that the pool selects it. */
  serviceLabels: Record<string, string>;
}

export interface SettingsPoolFamily {
  available: number;
  family: "IPv4" | "IPv6";
  total: number;
  used: number;
}

export interface SettingsStorageClass {
  allowVolumeExpansion: boolean;
  /** IsDefault is true for the cluster default storage class. */
  isDefault: boolean;
  /** @example "longhorn" */
  name: string;
  /** @example "driver.longhorn.io" */
  provisioner: string;
  /** @example "Retain" */
  reclaimPolicy: string;
  /** @example "Immediate" */
  volumeBindingMode: string;
}

export interface V1Alpha1ConfigFile {
  /** File is the path relative to the server root (/home/container). */
  file: string;
  /** +kubebuilder:validation:Enum=file;yaml;yml;properties;ini;json;xml */
  parser: string;
  replace?: V1Alpha1ConfigReplace[];
}

export interface V1Alpha1ConfigReplace {
  /**
   * IfValue only replaces when the current value equals it; a "regex:" prefix
   * turns it into a regular expression replacement on the current value.
   * +optional
   */
  ifValue?: string;
  /** Match is the key (or line prefix for the "file" parser) to look for. */
  match: string;
  /** ReplaceWith is the new value; placeholders like {{server.build.default.port}} are supported. */
  replaceWith: string;
  /**
   * ValueType is the JSON type of the value in the egg (string, number or boolean).
   * +kubebuilder:validation:Enum=string;number;boolean
   * +optional
   */
  valueType?: string;
}

export interface V1Alpha1DockerImage {
  image: string;
  name: string;
}

export interface V1Alpha1Egg {
  /**
   * APIVersion defines the versioned schema of this representation of an object.
   * Servers should convert recognized schemas to the latest internal value, and
   * may reject unrecognized values.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
   * +optional
   */
  apiVersion?: string;
  /**
   * Kind is a string value representing the REST resource this object represents.
   * Servers may infer this from the endpoint the client submits requests to.
   * Cannot be updated.
   * In CamelCase.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
   * +optional
   */
  kind?: string;
  metadata?: V1ObjectMeta;
  spec: V1Alpha1EggSpec;
  /** +optional */
  status?: V1Alpha1EggStatus;
}

export interface V1Alpha1EggSource {
  /**
   * AutoUpdate downloads the update URL every hour and replaces the egg when the file changed
   * (changes made in the panel are replaced, like "Update from URL" does).
   * +optional
   */
  autoUpdate?: boolean;
  /**
   * EditedAt is set when the egg was changed in the panel after its import.
   * +optional
   */
  editedAt?: string;
  /**
   * ExportedAt is the exported_at of the imported file: when its panel exported the egg.
   * +optional
   */
  exportedAt?: string;
  /** Format of the imported file, e.g. PTDL_v2 or PLCN_v3. */
  format?: string;
  importedAt?: string;
  /** ImportedFrom is the URL the egg was fetched from, if any. */
  importedFrom?: string;
  updateUrl?: string;
  /**
   * UUID identifies the egg across panels (Pelican overwrites an egg with the same UUID on import).
   * +optional
   */
  uuid?: string;
}

export interface V1Alpha1EggSpec {
  author?: string;
  configFiles?: V1Alpha1ConfigFile[];
  description?: string;
  displayName: string;
  dockerImages: V1Alpha1DockerImage[];
  features?: string[];
  fileDenylist?: string[];
  /** Icon is an optional data URI (Pelican eggs). */
  icon?: string;
  install?: V1Alpha1InstallScript;
  source?: V1Alpha1EggSource;
  /** Startup is the default startup command (with {{VAR}} placeholders): the first of StartupCommands. */
  startup: string;
  /**
   * StartupCommands are the startup commands owners pick from per server; the first is the default.
   * Empty on eggs saved before it existed: then Startup is the only one.
   * +optional
   */
  startupCommands?: V1Alpha1StartupCommand[];
  /**
   * StartupDone lists console output snippets that mark the server as running.
   * A "regex:" prefix makes an entry a regular expression.
   */
  startupDone?: string[];
  /** Stop is the stop command; a leading "^" means a signal (^C = SIGINT, ^^C = SIGKILL). */
  stop?: string;
  stripAnsi?: boolean;
  /**
   * Tags group eggs (Pelican).
   * +optional
   */
  tags?: string[];
  variables?: V1Alpha1EggVariable[];
}

export interface V1Alpha1EggStatus {
  /**
   * UpdateCheckedAt is the last automatic check of the update URL.
   * +optional
   */
  updateCheckedAt?: string;
  /**
   * UpdateError is the problem of the last automatic check ("" when it worked).
   * +optional
   */
  updateError?: string;
  /**
   * UpdatedAt is when the last automatic check found a newer file and applied it.
   * +optional
   */
  updatedAt?: string;
}

export interface V1Alpha1EggVariable {
  defaultValue?: string;
  description?: string;
  envVariable: string;
  fieldType?: string;
  name: string;
  /** Rules are Laravel style validation rules, e.g. "required|string|max:20". */
  rules?: string;
  userEditable?: boolean;
  userViewable?: boolean;
}

export interface V1Alpha1GameServer {
  /**
   * APIVersion defines the versioned schema of this representation of an object.
   * Servers should convert recognized schemas to the latest internal value, and
   * may reject unrecognized values.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
   * +optional
   */
  apiVersion?: string;
  /**
   * Kind is a string value representing the REST resource this object represents.
   * Servers may infer this from the endpoint the client submits requests to.
   * Cannot be updated.
   * In CamelCase.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
   * +optional
   */
  kind?: string;
  metadata?: V1ObjectMeta;
  spec: V1Alpha1GameServerSpec;
  status?: V1Alpha1GameServerStatus;
}

export interface V1Alpha1GameServerSpec {
  /**
   * IPv6 asks for an address of each IP family the cluster has (PreferDualStack); the
   * server gets an IPv6 address when the cluster runs dual stack and the pool has an IPv6 block.
   * +optional
   */
  ipv6?: boolean;
  /**
   * CrashRestart restarts the server after a crash (not twice within 60 seconds).
   * +kubebuilder:default=true
   */
  crashRestart?: boolean;
  /** DisplayName is a human friendly name. */
  displayName?: string;
  /** EggRef is the name of the Egg in the same namespace. */
  eggRef: string;
  /**
   * Environment holds values for the egg variables.
   * +optional
   */
  environment?: Record<string, string>;
  /**
   * ExternalTrafficPolicy of the service; empty means Local.
   * +optional
   */
  externalTrafficPolicy?: V1Alpha1TrafficPolicy;
  /** Image is the runtime image (one of the egg's docker images or a custom one). */
  image: string;
  /**
   * InstallRevision triggers a (re)install whenever it differs from status.installedRevision.
   * +kubebuilder:default=1
   */
  installRevision?: number;
  /**
   * LoadBalancerIP requests specific IPs from the load balancer pool: one, or one per IP family
   * separated by a comma. The server then gets only these addresses.
   * +optional
   */
  loadBalancerIP?: string;
  /**
   * LoadBalancerPool is the Cilium LB IPAM pool the address comes from. The service
   * gets the labels of the pool's service selector.
   * +optional
   */
  loadBalancerPool?: string;
  /**
   * Ports are published as TCP and UDP. The first one is the primary port (SERVER_PORT).
   * +kubebuilder:validation:MinItems=1
   */
  ports: number[];
  resources: V1Alpha1Resources;
  /**
   * Schedules run tasks at fixed times.
   * +optional
   * +listType=map
   * +listMapKey=name
   * +kubebuilder:validation:MaxItems=20
   */
  schedules?: V1Alpha1Schedule[];
  /**
   * SkipInstall skips the egg install script.
   * +optional
   */
  skipInstall?: boolean;
  /**
   * Startup overrides the egg's startup command.
   * +optional
   */
  startup?: string;
  /**
   * StartupName picks one of the egg's startup commands by name; empty or a name the egg no longer has
   * means its default. Startup wins when set.
   * +optional
   */
  startupName?: string;
  /**
   * State is the desired power state.
   * +kubebuilder:default=Stopped
   */
  state?: V1Alpha1PowerState;
  /**
   * StopTimeoutSeconds is how long to wait after the stop command before killing the server.
   * +kubebuilder:default=600
   */
  stopTimeoutSeconds?: number;
  /**
   * StorageClass of the data volume; it cannot be changed once the volume exists.
   * +optional
   * +kubebuilder:validation:XValidation:rule="self == oldSelf",message="storageClass cannot be changed"
   */
  storageClass?: string;
  /**
   * Suspended servers are stopped and cannot be started or used by their owner (admins only).
   * +optional
   */
  suspended?: boolean;
}

export interface V1Alpha1GameServerStatus {
  /** Address is the external IP assigned by the load balancer (the first of Addresses). */
  address?: string;
  /**
   * Addresses are all external IPs assigned by the load balancer (one per IP family).
   * +optional
   */
  addresses?: string[];
  diskMeasuredAt?: string;
  /**
   * DiskUsedBytes is the last measured size of the server files; the files pod that
   * measures it only runs on demand.
   */
  diskUsedBytes?: number;
  /** InstallExitCode is the exit code of the last install script run. */
  installExitCode?: number;
  /** InstalledRevision is the last completed install revision. */
  installedRevision?: number;
  /** LastCrashAt is the time of the last detected crash. */
  lastCrashAt?: string;
  /** LastExitCode is the exit code of the last terminated game process. */
  lastExitCode?: number;
  message?: string;
  observedGeneration?: number;
  phase?: V1Alpha1Phase;
  /** PodUID identifies the current game pod. */
  podUID?: string;
  /**
   * RestartRequired is set while the running game pod was created with other runtime
   * settings (image, startup, variables, memory, CPU, ports) than the spec has now.
   */
  restartRequired?: boolean;
  /**
   * Schedules holds the last run of every schedule.
   * +optional
   * +listType=map
   * +listMapKey=name
   */
  schedules?: V1Alpha1ScheduleStatus[];
  /** StartedAt is when the current game pod was created. */
  startedAt?: string;
  /** StopRequestedAt is set when a graceful stop was sent to the current pod. */
  stopRequestedAt?: string;
  /**
   * StopTasksStartedAt is set while the "stopping" schedules of the current pod run; the
   * stop is sent when they are done.
   * +optional
   */
  stopTasksStartedAt?: string;
}

export interface V1Alpha1InstallScript {
  container?: string;
  entrypoint?: string;
  script?: string;
}

export interface V1Alpha1OIDCIdentity {
  issuer: string;
  subject: string;
}

export interface V1Alpha1OIDCSettings {
  /**
   * Members of AdminGroup sign in as administrators, members of UserGroup as users; nobody
   * else may sign in.
   * +optional
   */
  adminGroup?: string;
  /** +optional */
  clientId?: string;
  /** +optional */
  enabled?: boolean;
  /**
   * GroupsClaim holds the groups of the user (default groups).
   * +optional
   */
  groupsClaim?: string;
  /**
   * IssuerURL is the issuer of the identity provider (its discovery document is at
   * <issuerUrl>/.well-known/openid-configuration).
   * +optional
   */
  issuerUrl?: string;
  /**
   * KeepPasswords keeps the password of an existing account when it is linked to the identity
   * provider; otherwise the account then signs in only through the identity provider.
   * +optional
   */
  keepPasswords?: boolean;
  /**
   * LinkByUsername links an existing account that is not linked yet to the user of the identity
   * provider with the same username (to bootstrap). Off: such a sign-in is refused, so nobody
   * can take over an account by choosing its name at the identity provider.
   * +optional
   */
  linkByUsername?: boolean;
  /**
   * Name is shown on the sign-in button: "Sign in with <name>".
   * +optional
   * +kubebuilder:validation:MaxLength=40
   */
  name?: string;
  /**
   * RedirectURL is the callback of the panel registered at the identity provider
   * (https://<panel>/api/auth/oidc/callback). It is not taken from the request: a forged Host
   * header must not send the code elsewhere.
   * +optional
   */
  redirectUrl?: string;
  /** +optional */
  userGroup?: string;
  /**
   * UsernameClaim holds the panel username (default preferred_username).
   * +optional
   */
  usernameClaim?: string;
}

export type V1Alpha1Phase =
  | "Pending"
  | "Installing"
  | "InstallFailed"
  | "Offline"
  | "Starting"
  | "Running"
  | "Stopping";

export type V1Alpha1PowerState = "Running" | "Stopped";

export interface V1Alpha1Resources {
  /**
   * CPUMillis limits CPU usage (1000 = one core); 0 means unlimited.
   * +optional
   */
  cpuMillis?: number;
  /**
   * DiskMiB is the size of the persistent volume.
   * +kubebuilder:validation:Minimum=256
   */
  diskMiB: number;
  /**
   * MemoryMiB is the memory available to the server (SERVER_MEMORY).
   * +kubebuilder:validation:Minimum=64
   */
  memoryMiB: number;
}

export interface V1Alpha1Schedule {
  /**
   * Cron is a five-field cron expression ("0 4 * * *") or a descriptor like "@daily",
   * evaluated in the panel's time zone. Either Cron or Event is set.
   * +optional
   */
  cron?: string;
  /** Enabled schedules run automatically; disabled ones can still be run by hand. */
  enabled: boolean;
  /**
   * Event runs the schedule when the server was marked as running ("started") or before it
   * is stopped or restarted ("stopping"; the stop waits until the tasks are done).
   * +optional
   * +kubebuilder:validation:Enum=started;stopping
   */
  event?: "started" | "stopping";
  /**
   * Name identifies the schedule within the server.
   * +kubebuilder:validation:MinLength=1
   * +kubebuilder:validation:MaxLength=50
   */
  name: string;
  /**
   * OnlyWhenOnline skips the run when the server is not running.
   * +optional
   */
  onlyWhenOnline?: boolean;
  /**
   * +kubebuilder:validation:MinItems=1
   * +kubebuilder:validation:MaxItems=10
   */
  tasks: V1Alpha1ScheduleTask[];
}

export interface V1Alpha1ScheduleStatus {
  /** LastResult is "ok", "skipped: …" or "failed: …". */
  lastResult?: string;
  lastRunAt?: string;
  name: string;
}

export interface V1Alpha1ScheduleTask {
  /**
   * Action is a console command, a power action or a backup.
   * +kubebuilder:validation:Enum=command;start;stop;restart;kill;backup
   */
  action: "command" | "start" | "stop" | "restart" | "kill" | "backup";
  /**
   * DelaySeconds waits before this task (after the previous one).
   * +optional
   * +kubebuilder:validation:Minimum=0
   * +kubebuilder:validation:Maximum=900
   */
  delaySeconds?: number;
  /**
   * Payload is the console command (action "command") or the backup label (action "backup").
   * +optional
   * +kubebuilder:validation:MaxLength=500
   */
  payload?: string;
}

export interface V1Alpha1StartupCommand {
  command: string;
  name: string;
}

export type V1Alpha1TrafficPolicy = "Local" | "Cluster";

export type V1Alpha1UserRole = "admin" | "user";

export type V1FieldsV1 = object;

export interface V1ManagedFieldsEntry {
  /**
   * FieldsV1 holds the first JSON version format as described in the "FieldsV1" type.
   * +optional
   */
  fieldsV1?: V1FieldsV1;
  /**
   * APIVersion defines the version of this resource that this field set
   * applies to. The format is "group/version" just like the top-level
   * APIVersion field. It is necessary to track the version of a field
   * set because it cannot be automatically converted.
   */
  apiVersion?: string;
  /**
   * FieldsType is the discriminator for the different fields format and version.
   * There is currently only one possible value: "FieldsV1"
   */
  fieldsType?: string;
  /** Manager is an identifier of the workflow managing these fields. */
  manager?: string;
  /**
   * Operation is the type of operation which lead to this ManagedFieldsEntry being created.
   * The only valid values for this field are 'Apply' and 'Update'.
   * +k8s:alpha(since: "1.37")=+k8s:required
   */
  operation?: V1ManagedFieldsOperationType;
  /**
   * Subresource is the name of the subresource used to update that object, or
   * empty string if the object was updated through the main resource. The
   * value of this field is used to distinguish between managers, even if they
   * share the same name. For example, a status update will be distinct from a
   * regular update using the same manager name.
   * Note that the APIVersion field is not related to the Subresource field and
   * it always corresponds to the version of the main resource.
   */
  subresource?: string;
  /**
   * Time is the timestamp of when the ManagedFields entry was added. The
   * timestamp will also be updated if a field is added, the manager
   * changes any of the owned fields value or removes a field. The
   * timestamp does not update when a field is removed from the entry
   * because another manager took it over.
   * +optional
   */
  time?: string;
}

export type V1ManagedFieldsOperationType = "Apply" | "Update";

export interface V1ObjectMeta {
  /**
   * Annotations is an unstructured key value map stored with a resource that may be
   * set by external tools to store and retrieve arbitrary metadata. They are not
   * queryable and should be preserved when modifying objects.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations
   * +optional
   */
  annotations?: Record<string, string>;
  /**
   * CreationTimestamp is a timestamp representing the server time when this object was
   * created. It is not guaranteed to be set in happens-before order across separate operations.
   * Clients may not set this value. It is represented in RFC3339 form and is in UTC.
   *
   * Populated by the system.
   * Read-only.
   * Null for lists.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
   * +optional
   * +k8s:alpha(since: "1.37")=+k8s:immutable
   */
  creationTimestamp?: string;
  /**
   * Number of seconds allowed for this object to gracefully terminate before
   * it will be removed from the system. Only set when deletionTimestamp is also set.
   * May only be shortened.
   * Read-only.
   * +optional
   * +k8s:alpha(since: "1.37")=+k8s:optional
   * +k8s:alpha(since: "1.37")=+k8s:immutable
   */
  deletionGracePeriodSeconds?: number;
  /**
   * DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This
   * field is set by the server when a graceful deletion is requested by the user, and is not
   * directly settable by a client. The resource is expected to be deleted (no longer visible
   * from resource lists, and not reachable by name) after the time in this field, once the
   * finalizers list is empty. As long as the finalizers list contains items, deletion is blocked.
   * Once the deletionTimestamp is set, this value may not be unset or be set further into the
   * future, although it may be shortened or the resource may be deleted prior to this time.
   * For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react
   * by sending a graceful termination signal to the containers in the pod. After that 30 seconds,
   * the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup,
   * remove the pod from the API. In the presence of network partitions, this object may still
   * exist after this timestamp, until an administrator or automated process can determine the
   * resource is fully terminated.
   * If not set, graceful deletion of the object has not been requested.
   *
   * Populated by the system when a graceful deletion is requested.
   * Read-only.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
   * +optional
   * +k8s:alpha(since: "1.37")=+k8s:optional
   * +k8s:alpha(since: "1.37")=+k8s:immutable
   */
  deletionTimestamp?: string;
  /**
   * Must be empty before the object is deleted from the registry. Each entry
   * is an identifier for the responsible component that will remove the entry
   * from the list. If the deletionTimestamp of the object is non-nil, entries
   * in this list can only be removed.
   * Finalizers may be processed and removed in any order.  Order is NOT enforced
   * because it introduces significant risk of stuck finalizers.
   * finalizers is a shared field, any actor with permission can reorder it.
   * If the finalizer list is processed in order, then this can lead to a situation
   * in which the component responsible for the first finalizer in the list is
   * waiting for a signal (field value, external system, or other) produced by a
   * component responsible for a finalizer later in the list, resulting in a deadlock.
   * Without enforced ordering finalizers are free to order amongst themselves and
   * are not vulnerable to ordering changes in the list.
   * +optional
   * +patchStrategy=merge
   * +listType=set
   */
  finalizers?: string[];
  /**
   * GenerateName is an optional prefix, used by the server, to generate a unique
   * name ONLY IF the Name field has not been provided.
   * If this field is used, the name returned to the client will be different
   * than the name passed. This value will also be combined with a unique suffix.
   * The provided value has the same validation rules as the Name field,
   * and may be truncated by the length of the suffix required to make the value
   * unique on the server.
   *
   * If this field is specified and the generated name exists, the server will return a 409.
   *
   * Applied only if Name is not specified.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency
   * +optional
   */
  generateName?: string;
  /**
   * A sequence number representing a specific generation of the desired state.
   * Populated by the system. Read-only.
   * +optional
   * +k8s:alpha(since: "1.37")=+k8s:optional
   * +k8s:alpha(since: "1.37")=+k8s:minimum=0
   */
  generation?: number;
  /**
   * Map of string keys and values that can be used to organize and categorize
   * (scope and select) objects. May match selectors of replication controllers
   * and services.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels
   * +optional
   */
  labels?: Record<string, string>;
  /**
   * ManagedFields maps workflow-id and version to the set of fields
   * that are managed by that workflow. This is mostly for internal
   * housekeeping, and users typically shouldn't need to set or
   * understand this field. A workflow can be the user's name, a
   * controller's name, or the name of a specific apply path like
   * "ci-cd". The set of fields is always in the version that the
   * workflow used when modifying the object.
   *
   * +optional
   * +listType=atomic
   * +k8s:alpha(since: "1.37")=+k8s:optional
   */
  managedFields?: V1ManagedFieldsEntry[];
  /**
   * Name must be unique within a namespace. Is required when creating resources, although
   * some resources may allow a client to request the generation of an appropriate name
   * automatically. Name is primarily intended for creation idempotence and configuration
   * definition.
   * Cannot be updated.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names
   * +optional
   */
  name?: string;
  /**
   * Namespace defines the space within which each name must be unique. An empty namespace is
   * equivalent to the "default" namespace, but "default" is the canonical representation.
   * Not all objects are required to be scoped to a namespace - the value of this field for
   * those objects will be empty.
   *
   * Must be a DNS_LABEL.
   * Cannot be updated.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces
   * +optional
   */
  namespace?: string;
  /**
   * List of objects depended by this object. If ALL objects in the list have
   * been deleted, this object will be garbage collected. If this object is managed by a controller,
   * then an entry in this list will point to this controller, with the controller field set to true.
   * There cannot be more than one managing controller.
   * +optional
   * +patchMergeKey=uid
   * +patchStrategy=merge
   * +listType=map
   * +listMapKey=uid
   * +k8s:alpha(since:"1.37")=+k8s:optional
   */
  ownerReferences?: V1OwnerReference[];
  /**
   * An opaque value that represents the internal version of this object that can
   * be used by clients to determine when objects have changed. May be used for optimistic
   * concurrency, change detection, and the watch operation on a resource or set of resources.
   * Clients must treat these values as opaque and passed unmodified back to the server.
   * They may only be valid for a particular resource or set of resources.
   *
   * Populated by the system.
   * Read-only.
   * Value must be treated as opaque by clients and .
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
   * +optional
   */
  resourceVersion?: string;
  /**
   * Deprecated: selfLink is a legacy read-only field that is no longer populated by the system.
   * +optional
   */
  selfLink?: string;
  /**
   * UID is the unique in time and space value for this object. It is typically generated by
   * the server on successful creation of a resource and is not allowed to change on PUT
   * operations.
   *
   * Populated by the system.
   * Read-only.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids
   * +optional
   * +k8s:alpha(since: "1.37")=+k8s:optional
   * +k8s:alpha(since: "1.37")=+k8s:immutable
   */
  uid?: string;
}

export interface V1OwnerReference {
  /**
   * API version of the referent.
   * +k8s:alpha(since:"1.37")=+k8s:required
   */
  apiVersion: string;
  /**
   * If true, AND if the owner has the "foregroundDeletion" finalizer, then
   * the owner cannot be deleted from the key-value store until this
   * reference is removed.
   * See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion
   * for how the garbage collector interacts with this field and enforces the foreground deletion.
   * Defaults to false.
   * To set this field, a user needs "delete" permission of the owner,
   * otherwise 422 (Unprocessable Entity) will be returned.
   * +optional
   */
  blockOwnerDeletion?: boolean;
  /**
   * If true, this reference points to the managing controller.
   * +optional
   */
  controller?: boolean;
  /**
   * Kind of the referent.
   * More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
   * +k8s:alpha(since:"1.37")=+k8s:required
   */
  kind: string;
  /**
   * Name of the referent.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names
   * +k8s:alpha(since:"1.37")=+k8s:required
   */
  name: string;
  /**
   * UID of the referent.
   * More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids
   * +k8s:alpha(since:"1.37")=+k8s:required
   */
  uid: string;
}
