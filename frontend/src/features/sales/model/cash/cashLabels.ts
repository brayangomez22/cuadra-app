import type { components } from '@/shared/api/schema';

export type CashMovementKind = components['schemas']['CashMovementKind'];
export type CashResult = components['schemas']['CashClosing']['result'];

export const movementLabels: Record<CashMovementKind, string> = {
  in: 'Ingreso',
  out: 'Egreso',
};

export const resultLabels: Record<CashResult, string> = {
  exact: 'Cuadra',
  shortage: 'Faltante',
  overage: 'Sobrante',
};

/** The badge of the count's result: short is the one to act on. */
export const resultTones: Record<CashResult, 'success' | 'danger' | 'warning'> = {
  exact: 'success',
  shortage: 'danger',
  overage: 'warning',
};
