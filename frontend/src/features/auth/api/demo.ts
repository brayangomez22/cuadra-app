import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { ApiError, unwrap } from '@/shared/api/errors';
import { demosQueryKey } from './keys';
import type { AuthSession, DemoOption, DemoRequest } from './types';

/**
 * The business types with a demo (GET /auth/demo). The demo off (404) is no
 * demo at all: the login just does not offer it.
 */
export function useDemos() {
  return useQuery({
    queryKey: demosQueryKey,
    queryFn: async (): Promise<DemoOption[]> => {
      const { data, error, response } = await api.GET('/api/v1/auth/demo');
      if (data) return data.items;
      if (response.status === 404) return [];
      throw new ApiError(response.status, error);
    },
    staleTime: Infinity,
  });
}

/** POST /auth/demo: the session of the chosen role in the demo. */
export async function startDemo(body: DemoRequest): Promise<AuthSession> {
  return unwrap(await api.POST('/api/v1/auth/demo', { body }));
}
