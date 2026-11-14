// What this app gives the base besides the navigation (app/use-navigation.ts) and its parts of the users and settings
// pages (app/users.tsx, app/settings.ts): its names and brand, roles, texts of shared pages, the links and request
// rates of the footer, the names in the breadcrumbs and the data the base's changes affect. Plain values, also used
// on pages without a signed-in user.

import { ArrowDownUpIcon, BugIcon, ContainerIcon, GlobeCodeIcon, ServerPlusIcon } from "lucide-react"

import type { PanelSettings, Role } from "@/base/lib/types"
import type { Crumb, FooterLink, RateItem } from "@/base/types"
import { serverName } from "@/lib/format"
import { useEgg, useServer } from "@/lib/queries"
import { keys } from "@/lib/queries/keys"

/** The product name (setup page, placeholders) and its id (chart, client id and group examples). */
export const appName = "Kubedactyl"
export const appId = "kubedactyl"

/** The tagline the branding shows unless the settings name another (backend app.BrandTagline). */
export const appTagline = "Game servers on Kubernetes"

/** Prefix of the panel's environment variables (KUBEDACTYL_SETUP_TOKEN). */
export const envPrefix = "KUBEDACTYL"

/** Prefix of personal access tokens (backend app.APITokenPrefix). */
export const apiTokenPrefix = "kdt_"

/** The roles, the default one first: new users and invites get it, single sign-on maps memberGroup to it. */
export const roles: { value: Role; label: string }[] = [
  { value: "user", label: "User" },
  { value: "admin", label: "Administrator" },
]

/** The single sign-on setting with the group of the default role. */
export const memberGroup = "userGroup" satisfies keyof NonNullable<PanelSettings["oidc"]>

/** Where the first administrator goes after the setup, and what to do there. */
export const setupNext = { to: "/settings", hint: "Configure the external domain, storage classes and pools next." }

/** What keeps running while the panel upgrades itself. */
export const upgradeNote = "Game servers keep running."

/** Colors of the light rays and particles behind the sign-in page (the brand's light tones). */
export const authColors = { rays: "rgba(16, 185, 129, 0.22)", light: "#059669", dark: "#6ee7b7" }

/** Data a saved setting changes besides the base's own: health checks, server addresses, the egg library. */
export const settingsDependents = [keys.cluster, keys.clusterHealth, keys.servers, keys.library]

/** Data that goes with a deleted user: their servers. */
export const userDependents = [keys.servers]

/** The brand gradient (emerald to sky) of the built-in logo, the favicon, GlowCard and the avatars. */
export const BRAND_FROM = "#00bc7d"
export const BRAND_TO = "#0084d1"

/** The icon of the built-in logo. */
export const brandIcon = { icon: ContainerIcon }

/** The crumb of the home page; the browser tab leaves it out on other pages. */
export const homeLabel = "Dashboard"

const REPOSITORY_URL = "https://github.com/Syntax3rror404/kubedactyl"

/** Links in the footer, after the version. */
export const footerLinks: FooterLink[] = [
  { label: "GitHub", href: REPOSITORY_URL, icon: GlobeCodeIcon },
  { label: "Report an issue", href: `${REPOSITORY_URL}/issues/new`, icon: BugIcon },
]

/** The request rates in the footer: the user's own, then the ones only administrators get. */
export const rateItems: RateItem[] = [
  {
    key: "user",
    label: "User",
    icon: ArrowDownUpIcon,
    title: "Your requests per second over the last 5 seconds and the limit per user",
  },
  {
    key: "panel",
    label: "System",
    icon: ServerPlusIcon,
    title: "Kubernetes API calls per second of the whole panel over the last 5 seconds and the panel limit",
  },
]

/** The crumbs with display names: labels like "server:<name>" or "egg:<name>" resolve to the object's name. */
export function useCrumbLabels(crumbs: Crumb[]): Crumb[] {
  const ref = (prefix: string) => crumbs.find((c) => c.label.startsWith(prefix))?.label.slice(prefix.length) ?? ""
  const serverRef = ref("server:")
  const eggRef = ref("egg:")
  const server = useServer(serverRef)
  const egg = useEgg(eggRef)
  return crumbs.map((c) => {
    if (c.label.startsWith("server:")) return { ...c, label: server.data ? serverName(server.data) : serverRef }
    if (c.label.startsWith("egg:")) return { ...c, label: egg.data?.spec.displayName ?? eggRef }
    return c
  })
}
