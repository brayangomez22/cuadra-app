import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import type { InviteValues } from '../model/inviteSchema';
import { invitationsQueryKey } from './keys';
import type { Invitation } from './types';

/** The invitations neither accepted nor revoked, newest first. */
export function useInvitations() {
  return useQuery({
    queryKey: invitationsQueryKey,
    queryFn: async () => unwrap(await api.GET('/api/v1/invitations')),
  });
}

/** Invites someone; the answer carries the link's token, shown once. */
export function useInvite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: InviteValues) => unwrap(await api.POST('/api/v1/invitations', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: invitationsQueryKey }),
  });
}

/** Revokes an invitation: its link stops working. */
export function useRevokeInvitation() {
  const queryClient = useQueryClient();
  const toast = useMutationToast({ success: 'Invitación anulada', error: 'No se pudo anular la invitación' });
  return useMutation({
    mutationFn: async (invitation: Invitation) =>
      unwrapEmpty(
        await api.DELETE('/api/v1/invitations/{invitationId}', {
          params: { path: { invitationId: invitation.id } },
        }),
      ),
    onSuccess: async () => {
      toast.onSuccess(undefined);
      await queryClient.invalidateQueries({ queryKey: invitationsQueryKey });
    },
    onError: toast.onError,
  });
}
