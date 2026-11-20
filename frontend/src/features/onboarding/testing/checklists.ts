import type { components } from '@/shared/api/schema';

type Checklist = components['schemas']['OnboardingChecklist'];
type StepId = components['schemas']['OnboardingStepId'];

/** A checklist with steps; the ids listed in done are done. */
export function aChecklist(ids: StepId[], done: StepId[] = [], extra: Partial<Checklist> = {}): Checklist {
  const steps = ids.map((id) => ({ id, done: done.includes(id), critical: id === 'invoicing' }));
  return {
    visible: steps.some((s) => !s.done),
    dismissed: false,
    completed: steps.filter((s) => s.done).length,
    total: steps.length,
    steps,
    ...extra,
  };
}

/** The steps of a hardware store's template (with alternate units and credit customers). */
export const hardwareSteps: StepId[] = [
  'business_profile',
  'products',
  'alternate_units',
  'initial_stock',
  'customer_credit',
  'invite_team',
  'first_sale',
];

/** The steps of a clothing store's template (with variants). */
export const clothingSteps: StepId[] = [
  'business_profile',
  'products',
  'variant_products',
  'initial_stock',
  'invite_team',
  'first_sale',
];
