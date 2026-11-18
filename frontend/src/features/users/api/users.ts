import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import type { RoleValues } from '../model/inviteSchema';
import { usersQueryKey } from './keys';
import type { User } from './types';

/** The company's users, active or not, by name (GET /users, users:manage). */
export function useUsers() {
  return useQuery({
    queryKey: usersQueryKey,
    queryFn: async () => unwrap(await api.GET('/api/v1/users')),
  });
}

const path = (user: User) => ({ params: { path: { userId: user.id } } });

/** Gives a user another role; the list reloads before it resolves. */
export function useChangeRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ user, body }: { user: User; body: RoleValues }) =>
      unwrap(await api.PATCH('/api/v1/users/{userId}', { ...path(user), body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: usersQueryKey }),
  });
}

/** Deactivates (the user loses access at once) or reactivates a user. */
export function useSetUserActive() {
  const queryClient = useQueryClient();
  const toast = useMutationToast({
    success: (user: User) => ({
      title: user.active ? 'Usuario reactivado' : 'Usuario desactivado',
      description: user.name,
    }),
    error: 'No se pudo cambiar el usuario',
  });
  return useMutation({
    mutationFn: async ({ user, active }: { user: User; active: boolean }) =>
      unwrap(
        active
          ? await api.POST('/api/v1/users/{userId}/activate', path(user))
          : await api.POST('/api/v1/users/{userId}/deactivate', path(user)),
      ),
    onSuccess: async (user) => {
      toast.onSuccess(user);
      await queryClient.invalidateQueries({ queryKey: usersQueryKey });
    },
    onError: toast.onError,
  });
}

/** Issues a link for the user to choose a new password; only the last one works. */
export function useIssuePasswordReset() {
  return useMutation({
    mutationFn: async (user: User) => unwrap(await api.POST('/api/v1/users/{userId}/password-reset', path(user))),
    ...useMutationToast({ error: 'No se pudo generar el enlace' }),
  });
}
