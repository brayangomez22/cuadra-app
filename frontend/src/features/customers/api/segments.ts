import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { customersKeys } from './keys';

/** The tenant's customer segments, by name. Settings creates, renames and deletes them. */
export function useCustomerSegments() {
  return useQuery({
    queryKey: customersKeys.segments,
    queryFn: async () => unwrap(await api.GET('/api/v1/customer-segments')).items,
  });
}
