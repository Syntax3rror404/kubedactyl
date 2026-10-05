import { createBrowserRouter } from "react-router"

import { AppLayout } from "@/components/layout/app-layout"
import type { CrumbHandle } from "@/components/layout/site-header"
import { RequireAdmin, RequireAuth } from "@/features/auth/components/require-auth"

const crumbs = (fn: CrumbHandle["crumbs"]): CrumbHandle => ({ crumbs: fn })

// Pages are loaded on demand, so heavy editors (CodeMirror, xterm) are not in the first bundle.
const page =
  <M extends Record<string, React.ComponentType>>(load: () => Promise<M>, name: keyof M) =>
  async () => ({ Component: (await load())[name] })

/** Route table: which page (a *-page.tsx file in src/features) is shown for which URL. */
export const router = createBrowserRouter([
  { path: "/login", lazy: page(() => import("@/features/auth/login-page"), "LoginPage") },
  { path: "/setup", lazy: page(() => import("@/features/auth/setup-page"), "SetupPage") },
  { path: "/invite", lazy: page(() => import("@/features/auth/invite-page"), "InvitePage") },
  { path: "/licenses", lazy: page(() => import("@/features/licenses/licenses-page"), "LicensesPage") },
  {
    element: <RequireAuth />,
    children: [
      {
        path: "change-password",
        lazy: page(() => import("@/features/auth/change-password-page"), "ChangePasswordPage"),
      },
      {
        element: <AppLayout />,
        children: [
          {
            index: true,
            lazy: page(() => import("@/features/dashboard/dashboard-page"), "DashboardPage"),
            handle: crumbs(() => [{ label: "Dashboard" }]),
          },
          {
            path: "servers/:server",
            lazy: page(() => import("@/features/servers/server-layout"), "ServerLayout"),
            handle: crumbs((p) => [
              { label: "Dashboard", to: "/" },
              { label: `server:${p.server}`, to: `/servers/${p.server}` },
            ]),
            children: [
              {
                index: true,
                lazy: page(() => import("@/features/servers/console-page"), "ConsolePage"),
                handle: crumbs(() => [{ label: "Console" }]),
              },
              {
                path: "files",
                lazy: page(() => import("@/features/servers/files-page"), "FilesPage"),
                handle: crumbs(() => [{ label: "Files" }]),
              },
              {
                path: "startup",
                lazy: page(() => import("@/features/servers/startup-page"), "StartupPage"),
                handle: crumbs(() => [{ label: "Startup" }]),
              },
              {
                path: "schedules",
                lazy: page(() => import("@/features/servers/schedules-page"), "SchedulesPage"),
                handle: crumbs(() => [{ label: "Schedules" }]),
              },
              {
                path: "tasks",
                lazy: page(() => import("@/features/servers/tasks-page"), "TasksPage"),
                handle: crumbs(() => [{ label: "Tasks" }]),
              },
              {
                path: "backups",
                lazy: page(() => import("@/features/servers/backups-page"), "BackupsPage"),
                handle: crumbs(() => [{ label: "Backups" }]),
              },
              {
                path: "diagnostics",
                lazy: page(() => import("@/features/servers/diagnostics-page"), "DiagnosticsPage"),
                handle: crumbs(() => [{ label: "Diagnostics" }]),
              },
              {
                path: "settings",
                lazy: page(() => import("@/features/servers/settings-page"), "ServerSettingsPage"),
                handle: crumbs(() => [{ label: "Settings" }]),
              },
            ],
          },
          {
            path: "account",
            lazy: page(() => import("@/features/account/account-page"), "AccountPage"),
            handle: crumbs(() => [{ label: "Account" }]),
          },
          {
            // Administration
            element: <RequireAdmin />,
            children: [
              {
                path: "servers/new",
                lazy: page(() => import("@/features/servers/server-new-page"), "ServerNewPage"),
                handle: crumbs(() => [{ label: "Dashboard", to: "/" }, { label: "New server" }]),
              },
              {
                path: "eggs",
                lazy: page(() => import("@/features/eggs/eggs-page"), "EggsPage"),
                handle: crumbs(() => [{ label: "Eggs" }]),
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
                lazy: page(() => import("@/features/users/users-page"), "UsersPage"),
                handle: crumbs(() => [{ label: "Users" }]),
              },
              {
                path: "cluster",
                lazy: page(() => import("@/features/cluster/cluster-page"), "ClusterPage"),
                handle: crumbs(() => [{ label: "Cluster" }]),
              },
              {
                path: "settings",
                lazy: page(() => import("@/features/settings/settings-page"), "PanelSettingsPage"),
                handle: crumbs(() => [{ label: "Settings" }]),
              },
            ],
          },
          {
            path: "*",
            lazy: page(() => import("@/app/not-found-page"), "NotFoundPage"),
            handle: crumbs(() => [{ label: "Not found" }]),
          },
        ],
      },
    ],
  },
])
