import { pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { PAGE_SIZE, type Location } from '../../../api/types';
import { shownLocation } from '../../../model/shownLocation';

const locationParam = stringParam();
// Another location starts on its first page.
const resetPage = { reset: ['pagina'] };

/** The location shown (?sede=) and the page of its stock (?pagina=), kept in the URL. */
export function useStockLocation(locations: Location[]) {
  const [locationId, setLocationId] = useSearchParamsState('sede', locationParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  return {
    location: shownLocation(locations, locationId),
    chooseLocation: setLocationId,
    offset: (page - 1) * PAGE_SIZE,
    setOffset: (offset: number) => setPage(offset / PAGE_SIZE + 1),
  };
}
