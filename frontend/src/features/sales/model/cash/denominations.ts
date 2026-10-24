import { formatMoney } from '@/shared/format';
import type { components } from '@/shared/api/schema';

export type DenominationCount = components['schemas']['DenominationCount'];

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

export type CountRead = { counts: DenominationCount[]; invalid: Set<string> };

/**
 * Reads the count typed per denomination: what was counted (none of the empty
 * or zero ones) and the denominations whose text is not a whole number.
 */
export function readCounts(texts: Record<string, string>): CountRead {
  const counts: DenominationCount[] = [];
  const invalid = new Set<string>();
  for (const d of denominations) {
    const parsed = parseCount(texts[d.value] ?? '');
    if (!parsed.ok) invalid.add(d.value);
    else if (parsed.quantity > 0) counts.push({ denomination: d.value, quantity: parsed.quantity });
  }
  return { counts, invalid };
}

// The API answers denominations with decimals ("100000.00"); they are whole pesos.
function wholePesos(denomination: string): bigint {
  return BigInt(denomination.replace(/\.0+$/, ''));
}

/** The cash of one denomination, exact: 3 × $ 20.000 is "60000". */
export function denominationSubtotal({ denomination, quantity }: DenominationCount): string {
  return (wholePesos(denomination) * BigInt(quantity)).toString();
}

/** The cash counted, exact: denominations are whole pesos. */
export function countedTotal(counts: DenominationCount[]): string {
  return counts.reduce((sum, c) => sum + wholePesos(c.denomination) * BigInt(c.quantity), 0n).toString();
}
