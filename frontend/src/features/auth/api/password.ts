import { useMutation } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrapEmpty } from '@/shared/api/errors';
import type { ChangePasswordValues } from '../model/passwordSchemas';

/** POST /me/password: the user's own new password; the current one must match. */
export function useChangePassword() {
  return useMutation({
    mutationFn: async (body: ChangePasswordValues) => unwrapEmpty(await api.POST('/api/v1/me/password', { body })),
  });
}
