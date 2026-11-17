// Types of the base (src/base, the same in every app): what an app gives it in app/use-navigation.ts and
// app/config.ts and the states its components show.

import type { ComponentType } from "react"

/** Color of a status dot or count; pending: something is changing (starting, stopping). */
export type NavTone = "success" | "warning" | "pending" | "danger" | "info" | "muted"

/** Small sign at the end of a sidebar entry. */
export type NavBadge =
  /** A number in a colored pill (problems). */
  | { kind: "count"; value: number; tone: NavTone }
  /** A status dot, optionally with a short text before it ("5/6"). */
  | { kind: "status"; tone: NavTone; text?: string; pulse?: boolean }
  /** A newer version of the app: a pill with the version and a light circling its edge. */
  | { kind: "update"; version: string }
  /** Plain muted text (a number of objects). */
  | { kind: "text"; text: string }

/** One entry of the sidebar; with children it is a branch that opens while it holds the active page. */
export interface NavItem {
  label: string
  /** Page of the entry. Empty: the entry only opens its children. */
  to?: string
  /** Opens in a new tab (API documentation). */
  href?: string
  icon?: ComponentType
  /** Classes of a colored tile behind the icon ("text-sky-500 bg-sky-500/10"), so topics are told apart. */
  accent?: string
  /** The entry is the current page. */
  active?: boolean
  /** Monospace label (names of machines). */
  mono?: boolean
  badge?: NavBadge
  children?: NavItem[]
  /** Shown in an open branch without children ("No servers yet."). */
  empty?: string
  /** The branch starts open (the app's center), not only while it holds the current page. */
  open?: boolean
}

/** A part of the sidebar; without label the entries follow the one above directly. */
export interface NavGroup {
  label?: string
  items: NavItem[]
}

/** Entries the search finds besides the pages of the sidebar (objects, types). */
export interface NavSearchGroup {
  label: string
  items: { label: string; to: string; hint?: string }[]
}

/** One result of the short diagnosis in the header. */
export interface NavCheck {
  label: string
  message?: string
  tone: NavTone
}

export interface NavConfig {
  /**
   * Top of the sidebar: name and state of what the app manages, and its checks (failing ones always shown, the
   * others on click). Without checks it is the name, linking home.
   */
  header: {
    name: string
    detail?: string
    tone?: NavTone
    checks?: NavCheck[]
    /** The page with all details ("Diagnostics"). */
    checksLink?: { label: string; to: string }
  }
  groups: NavGroup[]
  /**
   * A group at the bottom, right above the user menu: the pages for administrators (users, settings) and links
   * like the API documentation. The user menu itself holds only the account.
   */
  footer?: NavGroup
  /** Second line of the user in the open menu when the user has no email. Empty: the role. */
  userDetail?: string
  search: NavSearchGroup[]
}

/** One step of the breadcrumbs; the last one is the current page. */
export interface Crumb {
  label: string
  to?: string
}

/** A link in the footer, opened in a new tab. */
export interface FooterLink {
  label: string
  href: string
  icon: ComponentType<{ className?: string }>
}

/** A request rate in the footer: `key` names the rate in the app's request rate answer ("user" first). */
export interface RateItem {
  key: string
  label: string
  icon: ComponentType<{ className?: string }>
  title: string
}

/** Result of a check (cluster health, diagnostics). */
export type CheckState = "ok" | "warning" | "error" | "info" | "skipped"

/** State of a background job (backups, upgrades). */
export type JobState = "running" | "done" | "succeeded" | "failed" | "cancelled"
