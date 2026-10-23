import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { businessTemplatesQueryKey } from './keys';

/** The business types a company can sign up with (GET /business-templates). */
export function useBusinessTemplates() {
  return useQuery({
    queryKey: businessTemplatesQueryKey,
    queryFn: async () => unwrap(await api.GET('/api/v1/business-templates')).items,
    staleTime: Infinity,
  });
}
