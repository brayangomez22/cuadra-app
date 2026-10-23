import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { catalogKeys } from './keys';

export function useCategories({ enabled = true } = {}) {
  return useQuery({
    queryKey: catalogKeys.categories,
    queryFn: async () => unwrap(await api.GET('/api/v1/categories')).items,
    enabled,
  });
}
