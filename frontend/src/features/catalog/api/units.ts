import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { catalogKeys } from './keys';

/** The tenant's units of measure, by name. */
export function useUnits() {
  return useQuery({
    queryKey: catalogKeys.units,
    queryFn: async () =>
      unwrap(await api.GET('/api/v1/units')).items.toSorted((a, b) => a.name.localeCompare(b.name, 'es')),
  });
}
