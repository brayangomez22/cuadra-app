import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { customersKeys } from './keys';

/** The DANE's municipalities, by department and name. The catalog does not change while the app runs. */
export function useMunicipalities() {
  return useQuery({
    queryKey: customersKeys.municipalities,
    queryFn: async () => unwrap(await api.GET('/api/v1/municipalities')).items,
    staleTime: Infinity,
  });
}
