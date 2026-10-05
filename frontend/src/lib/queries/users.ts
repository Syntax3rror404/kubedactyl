import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"
import { keys } from "@/lib/queries/keys"
import { refresh, useApiMutation, type MutationCallbacks } from "@/lib/queries/mutation"
import type {
  AcceptInviteRequest,
  CreatedInvite,
  CreateInviteRequest,
  CreateUserRequest,
  LoginResponse,
  UpdateUserRequest,
  UserView,
} from "@/lib/types"

export const useUsers = () => useQuery({ queryKey: keys.users, queryFn: api.users.list })

export const useCreateUser = (cb?: MutationCallbacks<UserView, CreateUserRequest>) =>
  useApiMutation(api.users.create, (qc) => refresh(qc, keys.users), cb)

export const useUpdateUser = (cb?: MutationCallbacks<UserView, { name: string; changes: UpdateUserRequest }>) =>
  useApiMutation(
    ({ name, changes }) => api.users.update(name, changes),
    (qc) => refresh(qc, keys.users),
    cb,
  )

/** Deleting a user also deletes their servers. */
export const useDeleteUser = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(api.users.delete, (qc) => refresh(qc, keys.users, keys.servers), cb)

export const useInvites = () => useQuery({ queryKey: keys.invites, queryFn: api.invites.list })

export const useCreateInvite = (cb?: MutationCallbacks<CreatedInvite, CreateInviteRequest>) =>
  useApiMutation(api.invites.create, (qc) => refresh(qc, keys.invites), cb)

export const useRenewInvite = (cb?: MutationCallbacks<CreatedInvite, string>) =>
  useApiMutation(api.invites.renew, (qc) => refresh(qc, keys.invites), cb)

export const useDeleteInvite = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(api.invites.delete, (qc) => refresh(qc, keys.invites), cb)

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
