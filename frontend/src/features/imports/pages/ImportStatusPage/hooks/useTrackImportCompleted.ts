import { useEffect, useRef } from 'react';
import { track } from '@/features/analytics';
import type { Import } from '../../../api/types';
import { importStage } from '../../../model/importProgress';

/**
 * Sends import_completed when the page sees the import go from running to
 * completed: opening the report of an old import again sends nothing.
 */
export function useTrackImportCompleted(found: Import | undefined) {
  const wasRunning = useRef(false);
  const stage = found ? importStage(found.status) : undefined;
  useEffect(() => {
    if (stage === 'running') wasRunning.current = true;
    if (stage !== 'completed' || !wasRunning.current || !found) return;
    wasRunning.current = false;
    void track('import_completed', { products: found.counts.products, errors: found.counts.errors });
  }, [stage, found]);
}
