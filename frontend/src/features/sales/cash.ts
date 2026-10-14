import { z } from 'zod';
import { formatMoney, isNegative, isZero, parseMoneyInput } from '@/shared/format';
import type { components } from '@/shared/api/schema';

export type DenominationCount = components['schemas']['DenominationCount'];
export type CashMovementKind = components['schemas']['CashMovementKind'];
export type CashResult = components['schemas']['CashClosing']['result'];

type Denomination = { value: string; kind: 'bill' | 'coin' | 'both' };

/**
 * The bills and coins of the Colombian peso the API accepts, from the largest.
 * The 1.000 bill and coin are counted together.
 */
export const denominations: Denomination[] = [
  { value: '100000', kind: 'bill' },
  { value: '50000', kind: 'bill' },
  { value: '20000', kind: 'bill' },
  { value: '10000', kind: 'bill' },
  { value: '5000', kind: 'bill' },
  { value: '2000', kind: 'bill' },
  { value: '1000', kind: 'both' },
  { value: '500', kind: 'coin' },
  { value: '200', kind: 'coin' },
  { value: '100', kind: 'coin' },
  { value: '50', kind: 'coin' },
];

const kindLabels: Record<Denomination['kind'], string> = {
  bill: 'Billetes',
  coin: 'Monedas',
  both: 'Billetes y monedas',
};

/** "Billetes de $ 50.000", the label of a denomination's field. */
export function denominationLabel({ value, kind }: Denomination): string {
  return `${kindLabels[kind]} de ${formatMoney(value)}`;
}

// The API's limit of bills or coins of one denomination.
const MAX_QUANTITY = 10_000_000;

export type CountInput = { ok: true; quantity: number } | { ok: false };

/** Reads how many bills or coins were typed: empty is none; only whole numbers. */
export function parseCount(text: string): CountInput {
  const trimmed = text.trim();
  if (trimmed === '') return { ok: true, quantity: 0 };
  if (!/^\d+$/.test(trimmed)) return { ok: false };
  const quantity = Number(trimmed);
  return quantity <= MAX_QUANTITY ? { ok: true, quantity } : { ok: false };
}

/** The cash counted, exact: denominations are whole pesos. */
export function countedTotal(counts: DenominationCount[]): string {
  return counts.reduce((sum, c) => sum + BigInt(c.denomination) * BigInt(c.quantity), 0n).toString();
}

export const movementLabels: Record<CashMovementKind, string> = {
  in: 'Ingreso',
  out: 'Egreso',
};

export const resultLabels: Record<CashResult, string> = {
  exact: 'Cuadra',
  shortage: 'Faltante',
  overage: 'Sobrante',
};

/**
 * An amount of pesos as a cashier types it ("150.000", "12500,50"): the form
 * gets the API's decimal string. min "positive" rejects zero.
 */
export function moneyField(min: 'positive' | 'nonnegative', empty: string) {
  return z
    .string()
    .superRefine((text, ctx) => {
      if (text.trim() === '') {
        ctx.addIssue({ code: 'custom', message: empty });
        return;
      }
      const parsed = parseMoneyInput(text);
      if (!parsed.ok || /\.\d{3,}$/.test(parsed.value)) {
        ctx.addIssue({
          code: 'custom',
          message: 'Escriba un valor, por ejemplo 50.000 o 50000',
        });
        return;
      }
      if (isNegative(parsed.value) || (min === 'positive' && isZero(parsed.value))) {
        ctx.addIssue({
          code: 'custom',
          message: min === 'positive' ? 'El valor debe ser mayor que cero' : 'El valor no puede ser negativo',
        });
      }
    })
    .transform((text) => {
      const parsed = parseMoneyInput(text);
      return parsed.ok ? parsed.value : text;
    });
}
