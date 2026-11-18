import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import type { AcceptInvitationValues } from '../model/passwordSchemas';
import { invitationQueryKey } from './keys';

/** What an invitation link offers before it is accepted; a used or expired link fails with its ApiError. */
export function useInvitationPreview(token: string | null) {
  return useQuery({
    queryKey: invitationQueryKey(token ?? ''),
    queryFn: async () => unwrap(await api.POST('/api/v1/auth/invitations/lookup', { body: { token: token ?? '' } })),
    enabled: token !== null,
  });
}

/** POST /auth/invitations/accept: the new user and their session. */
export async function acceptInvitation(token: string, values: AcceptInvitationValues) {
  return unwrap(await api.POST('/api/v1/auth/invitations/accept', { body: { token, ...values } }));
}

/** POST /auth/password-reset: the new password of a reset link. */
export function useResetPassword() {
  return useMutation({
    mutationFn: async ({ token, password }: { token: string; password: string }) =>
      unwrapEmpty(await api.POST('/api/v1/auth/password-reset', { body: { token, password } })),
  });
}
