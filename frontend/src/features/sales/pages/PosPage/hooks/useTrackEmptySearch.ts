import { useEffect } from 'react';
import { paths } from '@/app/routes/paths';
import { track } from '@/features/analytics';
import type { PosSearch } from './usePosSearch';

/**
 * Sends search_no_results once each time the POS search comes up empty
 * (by name or by a scanned code). What was searched is not sent.
 */
export function useTrackEmptySearch({ notFound, term, results }: PosSearch) {
  const empty = Boolean(notFound) || (Boolean(term) && results?.length === 0);
  useEffect(() => {
    if (empty) void track('search_no_results', { screen: paths.sales.pos() });
  }, [empty]);
}
