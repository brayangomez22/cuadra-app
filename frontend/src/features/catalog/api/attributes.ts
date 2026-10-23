import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { catalogKeys } from './keys';

/** The product attributes the tenant defines (brand, material...), in their order. */
export function useAttributeDefinitions() {
  return useQuery({
    queryKey: catalogKeys.attributes,
    queryFn: async () => unwrap(await api.GET('/api/v1/product-attributes')).items,
  });
}
