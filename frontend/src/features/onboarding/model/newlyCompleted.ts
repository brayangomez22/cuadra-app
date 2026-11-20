import type { ChecklistStep, StepId } from '../api/types';

/** The steps done that were not reported yet, in the checklist's order. */
export function newlyCompleted(steps: ChecklistStep[], reported: readonly string[]): StepId[] {
  return steps.filter((step) => step.done && !reported.includes(step.id)).map((step) => step.id);
}
