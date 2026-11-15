import { createBrowserRouter } from "react-router"

import { RequireAdmin, RequireAuth } from "@/base/features/auth/components/require-auth"
import { AppLayout } from "@/base/layout/app-layout"
import { ErrorPage } from "@/base/layout/error-page"
import type { CrumbHandle } from "@/base/layout/site-header"

const crumbs = (fn: CrumbHandle["crumbs"]): CrumbHandle => ({ crumbs: fn })
/** The handle of a page whose breadcrumb is only its own label. */
const titled = (label: string) => crumbs(() => [{ label }])

// Pages are loaded on demand, so heavy editors (CodeMirror, xterm) are not in the first bundle. A chunk of an older
// build resolves to nothing while main.tsx reloads the page; the route then waits for the reload instead of failing.
const page =
  <M extends Record<string, React.ComponentType>>(load: () => Promise<M>, name: keyof M) =>
  async () => {
    const loaded: M | undefined = await load()
    return loaded ? { Component: loaded[name] } : new Promise<never>(() => {})
  }

/** Route table: which page (a *-page.tsx file in src/features) is shown for which URL. */
export const router = createBrowserRouter([
  {
    errorElement: <ErrorPage />,
    children: [
      { path: "/login", lazy: page(() => import("@/base/features/auth/login-page"), "LoginPage") },
      { path: "/setup", lazy: page(() => import("@/base/features/auth/setup-page"), "SetupPage") },
      { path: "/invite", lazy: page(() => import("@/base/features/auth/invite-page"), "InvitePage") },
      { path: "/licenses", lazy: page(() => import("@/base/features/licenses/licenses-page"), "LicensesPage") },
      { path: "/api-docs", lazy: page(() => import("@/base/features/api-docs/api-docs-page"), "ApiDocsPage") },
      {
        element: <RequireAuth />,
        children: [
          {
            path: "change-password",
            lazy: page(() => import("@/base/features/auth/change-password-page"), "ChangePasswordPage"),
          },
          {
            element: <AppLayout />,
            children: [
              {
                index: true,
                lazy: page(() => import("@/features/dashboard/dashboard-page"), "DashboardPage"),
                handle: titled("Dashboard"),
              },
              {
                path: "servers/:server",
                lazy: page(() => import("@/features/servers/server-layout"), "ServerLayout"),
                handle: crumbs((p) => [
                  { label: "Servers" },
                  { label: `server:${p.server}`, to: `/servers/${p.server}` },
                ]),
                children: [
                  {
                    index: true,
                    lazy: page(() => import("@/features/servers/console-page"), "ConsolePage"),
                    handle: titled("Console"),
                  },
                  {
                    path: "files",
                    lazy: page(() => import("@/features/servers/files-page"), "FilesPage"),
                    handle: titled("Files"),
                  },
                  {
                    path: "startup",
                    lazy: page(() => import("@/features/servers/startup-page"), "StartupPage"),
                    handle: titled("Startup"),
                  },
                  {
                    path: "schedules",
                    lazy: page(() => import("@/features/servers/schedules-page"), "SchedulesPage"),
                    handle: titled("Schedules"),
                  },
                  {
                    path: "tasks",
                    lazy: page(() => import("@/features/servers/tasks-page"), "TasksPage"),
                    handle: titled("Tasks"),
                  },
                  {
                    path: "backups",
                    lazy: page(() => import("@/features/servers/backups-page"), "BackupsPage"),
                    handle: titled("Backups"),
                  },
                  {
                    path: "diagnostics",
                    lazy: page(() => import("@/features/servers/diagnostics-page"), "DiagnosticsPage"),
                    handle: titled("Diagnostics"),
                  },
                  {
                    path: "settings",
                    lazy: page(() => import("@/features/servers/settings-page"), "SettingsPage"),
                    handle: titled("Settings"),
                  },
                ],
              },
              {
                path: "account",
                lazy: page(() => import("@/base/features/account/account-page"), "AccountPage"),
                handle: titled("Account"),
              },
              {
                element: <RequireAdmin />,
                children: [
                  {
                    path: "servers/new",
                    lazy: page(() => import("@/features/servers/server-new-page"), "ServerNewPage"),
                    handle: crumbs(() => [{ label: "Servers" }, { label: "New server" }]),
                  },
                  {
                    path: "eggs",
                    lazy: page(() => import("@/features/eggs/eggs-page"), "EggsPage"),
                    handle: titled("Eggs"),
                  },
                  {
                    path: "eggs/new",
                    lazy: page(() => import("@/features/eggs/egg-new-page"), "EggNewPage"),
                    handle: crumbs(() => [{ label: "Eggs", to: "/eggs" }, { label: "New egg" }]),
                  },
                  {
                    path: "eggs/library/egg",
                    lazy: page(() => import("@/features/eggs/library-egg-page"), "LibraryEggPage"),
                    handle: crumbs(() => [
                      { label: "Eggs", to: "/eggs" },
                      { label: "Library", to: "/eggs?tab=library" },
                      { label: "Egg" },
                    ]),
                  },
                  {
                    path: "eggs/:egg/edit",
                    lazy: page(() => import("@/features/eggs/egg-edit-page"), "EggEditPage"),
                    handle: crumbs((p) => [
                      { label: "Eggs", to: "/eggs" },
                      { label: `egg:${p.egg}`, to: `/eggs/${p.egg}` },
                      { label: "Edit" },
                    ]),
                  },
                  {
                    path: "eggs/:egg",
                    lazy: page(() => import("@/features/eggs/egg-detail-page"), "EggDetailPage"),
                    handle: crumbs((p) => [{ label: "Eggs", to: "/eggs" }, { label: `egg:${p.egg}` }]),
                  },
                  {
                    path: "users",
                    lazy: page(() => import("@/base/features/users/users-page"), "UsersPage"),
                    handle: titled("Users"),
                  },
                  {
                    path: "cluster",
                    lazy: page(() => import("@/features/cluster/cluster-page"), "ClusterPage"),
                    handle: titled("Cluster"),
                  },
                  {
                    path: "settings",
                    lazy: page(() => import("@/base/features/settings/settings-page"), "PanelSettingsPage"),
                    handle: titled("Settings"),
                  },
                ],
              },
              {
                path: "*",
                lazy: page(() => import("@/base/layout/not-found-page"), "NotFoundPage"),
                handle: titled("Not found"),
              },
            ],
          },
        ],
      },
    ],
  },
])
