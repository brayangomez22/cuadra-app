import { useMutation, useQueryClient } from '@tanstack/react-query';
import { meQueryKey } from '@/features/auth';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

/** Changes the business's own settings; /me, which carries them, reloads before it resolves. */
export function useUpdateTenant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components['schemas']['TenantUpdate']) =>
      unwrap(await api.PATCH('/api/v1/tenant', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: meQueryKey }),
  });
}
