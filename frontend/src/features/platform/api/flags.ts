import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { meQueryKey } from '@/features/auth';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import { platformKeys } from './keys';

export function useFeatureFlags() {
  return useQuery({
    queryKey: platformKeys.flags,
    queryFn: async () => unwrap(await api.GET('/api/v1/platform/flags')).items,
  });
}

// Every change refreshes the list and /me: the operator's own business may be
// the one whose flags changed.
function useRefresh() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: platformKeys.flags }),
      queryClient.invalidateQueries({ queryKey: meQueryKey }),
    ]);
}

/** Turns a flag on or off for every business without an exception. */
export function useSetFlagGlobal() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: async ({ key, enabled }: { key: string; enabled: boolean }) =>
      unwrap(
        await api.PUT('/api/v1/platform/flags/{flagKey}', { params: { path: { flagKey: key } }, body: { enabled } }),
      ),
    onSuccess: refresh,
  });
}

type Exception = { key: string; tenantId: string };

/** Gives one business its own value of a flag. */
export function useSetFlagException() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: async ({ key, tenantId, enabled }: Exception & { enabled: boolean }) =>
      unwrapEmpty(
        await api.PUT('/api/v1/platform/flags/{flagKey}/tenants/{tenantId}', {
          params: { path: { flagKey: key, tenantId } },
          body: { enabled },
        }),
      ),
    onSuccess: refresh,
  });
}

/** Removes a business's exception: it follows the global value again. */
export function useClearFlagException() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: async ({ key, tenantId }: Exception) =>
      unwrapEmpty(
        await api.DELETE('/api/v1/platform/flags/{flagKey}/tenants/{tenantId}', {
          params: { path: { flagKey: key, tenantId } },
        }),
      ),
    onSuccess: refresh,
  });
}
