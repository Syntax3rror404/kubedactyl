import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refreshes, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type {
  AcceptInviteRequest,
  CreatedInvite,
  CreateInviteRequest,
  CreateUserRequest,
  LoginResponse,
  UpdateUserRequest,
  UserView,
} from "@/lib/types"

/** All accounts (admins) or the own subusers (users). */
export const useUsers = () => useQuery({ queryKey: keys.users, queryFn: api.users.list })

/** Creates an account. */
export const useCreateUser = (cb?: MutationCallbacks<UserView, CreateUserRequest>) =>
  useApiMutation(api.users.create, refreshes(keys.users), cb)

/** Changes an account. */
export const useUpdateUser = (cb?: MutationCallbacks<UserView, { name: string; changes: UpdateUserRequest }>) =>
  useApiMutation(({ name, changes }) => api.users.update(name, changes), refreshes(keys.users), cb)

/** Deleting a user also deletes their servers. */
export const useDeleteUser = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(api.users.delete, refreshes(keys.users, keys.servers), cb)

/** All invites (admins) or those of the own subusers (users), also expired ones. */
export const useInvites = () => useQuery({ queryKey: keys.invites, queryFn: api.invites.list })

/** Creates an invite; the answer holds the link, shown only once. */
export const useCreateInvite = (cb?: MutationCallbacks<CreatedInvite, CreateInviteRequest>) =>
  useApiMutation(api.invites.create, refreshes(keys.invites), cb)

/** Gives an invite a new link; the old one stops working. */
export const useRenewInvite = (cb?: MutationCallbacks<CreatedInvite, string>) =>
  useApiMutation(api.invites.renew, refreshes(keys.invites), cb)

/** Revokes an invite. */
export const useDeleteInvite = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(api.invites.delete, refreshes(keys.invites), cb)

/** Whether an invite link works (public; no retry: an invalid link stays invalid). */
export const useInvite = (token: string) =>
  useQuery({ queryKey: keys.invite(token), queryFn: () => api.invites.get(token), enabled: !!token, retry: false })

/** Creates the account of an invite link and signs it in; data of a previous account is dropped. */
export const useAcceptInvite = (cb?: MutationCallbacks<LoginResponse, AcceptInviteRequest>) =>
  useApiMutation(
    api.invites.accept,
    (qc, res) => {
      qc.clear()
      qc.setQueryData(keys.me, res.user)
    },
    cb,
  )
