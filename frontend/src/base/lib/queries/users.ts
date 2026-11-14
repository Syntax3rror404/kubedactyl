import { useQuery } from "@tanstack/react-query"

import { userDependents } from "@/app/config"
import { baseApi } from "@/base/lib/api"
import { baseKeys } from "@/base/lib/queries/keys"
import { refreshes, useApiMutation, type MutationCallbacks } from "@/base/lib/queries/mutation"
import type {
  AcceptInviteRequest,
  CreatedInvite,
  CreateInviteRequest,
  CreateUserRequest,
  LoginResponse,
  UpdateUserRequest,
  UserView,
} from "@/base/lib/types"

/** The accounts the user manages: all for administrators. */
export const useUsers = () => useQuery({ queryKey: baseKeys.users, queryFn: baseApi.users.list })

/** Creates an account. */
export const useCreateUser = (cb?: MutationCallbacks<UserView, CreateUserRequest>) =>
  useApiMutation(baseApi.users.create, refreshes(baseKeys.users), cb)

/** Changes an account. */
export const useUpdateUser = (cb?: MutationCallbacks<UserView, { name: string; changes: UpdateUserRequest }>) =>
  useApiMutation(({ name, changes }) => baseApi.users.update(name, changes), refreshes(baseKeys.users), cb)

/** Deletes an account and what the app removes with it (userDependents). */
export const useDeleteUser = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(baseApi.users.delete, refreshes(baseKeys.users, ...userDependents), cb)

/** The pending invites the user manages (all for administrators), also expired ones. */
export const useInvites = () => useQuery({ queryKey: baseKeys.invites, queryFn: baseApi.invites.list })

/** Creates an invite; the answer holds the link, shown only once. */
export const useCreateInvite = (cb?: MutationCallbacks<CreatedInvite, CreateInviteRequest>) =>
  useApiMutation(baseApi.invites.create, refreshes(baseKeys.invites), cb)

/** Gives an invite a new link; the old one stops working. */
export const useRenewInvite = (cb?: MutationCallbacks<CreatedInvite, string>) =>
  useApiMutation(baseApi.invites.renew, refreshes(baseKeys.invites), cb)

/** Revokes an invite. */
export const useDeleteInvite = (cb?: MutationCallbacks<void, string>) =>
  useApiMutation(baseApi.invites.delete, refreshes(baseKeys.invites), cb)

/** Whether an invite link works (public; no retry: an invalid link stays invalid). */
export const useInvite = (token: string) =>
  useQuery({
    queryKey: baseKeys.invite(token),
    queryFn: () => baseApi.invites.get(token),
    enabled: !!token,
    retry: false,
  })

/** Creates the account of an invite link and signs it in; data of a previous account is dropped. */
export const useAcceptInvite = (cb?: MutationCallbacks<LoginResponse, AcceptInviteRequest>) =>
  useApiMutation(
    baseApi.invites.accept,
    (qc, res) => {
      qc.clear()
      qc.setQueryData(baseKeys.me, res.user)
    },
    cb,
  )
