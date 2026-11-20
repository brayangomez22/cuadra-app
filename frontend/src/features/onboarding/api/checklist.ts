import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap, unwrapEmpty } from '@/shared/api/errors';
import { onboardingKeys } from './keys';
import type { Checklist } from './types';

/** The user's getting-started checklist, computed by the API from the tenant's data. */
export function useChecklist() {
  return useQuery({
    queryKey: onboardingKeys.checklist,
    queryFn: async () => unwrap(await api.GET('/api/v1/onboarding/checklist')),
  });
}

/**
 * Hides the checklist for the user. The API keeps what was pending, so the
 * list stays hidden until a critical step goes missing: the cache is set
 * hidden instead of read again.
 */
export function useDismissChecklist() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrapEmpty(await api.PUT('/api/v1/onboarding/checklist/dismissal')),
    onSuccess: () =>
      queryClient.setQueryData<Checklist>(onboardingKeys.checklist, (current) =>
        current ? { ...current, visible: false, dismissed: true } : current,
      ),
  });
}
