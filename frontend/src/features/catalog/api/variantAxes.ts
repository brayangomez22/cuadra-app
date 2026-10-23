import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { catalogKeys } from './keys';

/** The axes the tenant's variants vary on and their values, in their order. */
export function useVariantAxes({ enabled = true } = {}) {
  return useQuery({
    queryKey: catalogKeys.variantAxes,
    queryFn: async () => unwrap(await api.GET('/api/v1/variant-axes')).items,
    enabled,
  });
}
