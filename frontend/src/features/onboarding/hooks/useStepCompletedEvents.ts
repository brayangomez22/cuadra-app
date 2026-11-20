import { useEffect } from 'react';
import { track } from '@/features/analytics';
import type { ChecklistStep } from '../api/types';
import { newlyCompleted } from '../model/newlyCompleted';
import { loadReportedSteps, saveReportedSteps } from '../model/reportedSteps';

/**
 * Sends getting_started_step_completed once per step and tenant in this
 * browser, when the checklist first shows the step done. The steps are
 * computed from the tenant's data, so the event goes when someone opens the
 * home screen after completing it (docs/eventos.md).
 */
export function useStepCompletedEvents(tenantId: string, steps: ChecklistStep[] | undefined) {
  useEffect(() => {
    if (!steps || !tenantId) return;
    const reported = loadReportedSteps(tenantId);
    const fresh = newlyCompleted(steps, reported);
    if (fresh.length === 0) return;
    for (const step of fresh) void track('getting_started_step_completed', { step });
    saveReportedSteps(tenantId, [...reported, ...fresh]);
  }, [tenantId, steps]);
}
