import { pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { PAGE_SIZE } from '../../../api/types';

const idParam = stringParam();
// Changing a filter keeps the others and goes back to the first page.
const resetPage = { reset: ['pagina'] };

/** The variant (?variante=), the location (?sede=, none is every location) and the page of the kardex. */
export function useKardexFilters() {
  const [variantId, setVariantId] = useSearchParamsState('variante', idParam, resetPage);
  const [locationId, setLocationId] = useSearchParamsState('sede', idParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  return {
    variantId,
    locationId: locationId || undefined,
    chooseVariant: setVariantId,
    chooseLocation: (id: string | undefined) => setLocationId(id ?? ''),
    offset: (page - 1) * PAGE_SIZE,
    setOffset: (offset: number) => setPage(offset / PAGE_SIZE + 1),
  };
}
