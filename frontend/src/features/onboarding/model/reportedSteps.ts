// The steps whose completion this browser already reported to the
// analytics, per tenant: only step ids. Storage can be missing or blocked,
// so every access is guarded; without it a step may be reported again.

const keyOf = (tenantId: string) => `cuadra.onboarding.reported.${tenantId}`;

export function loadReportedSteps(tenantId: string): string[] {
  try {
    const stored = JSON.parse(localStorage.getItem(keyOf(tenantId)) ?? '[]') as unknown;
    return Array.isArray(stored) ? stored.filter((id): id is string => typeof id === 'string') : [];
  } catch {
    return [];
  }
}

export function saveReportedSteps(tenantId: string, steps: string[]) {
  try {
    localStorage.setItem(keyOf(tenantId), JSON.stringify(steps));
  } catch {
    // Without storage the next visit may report the step again.
  }
}
