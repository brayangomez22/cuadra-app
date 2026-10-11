// Exact arithmetic for amounts and quantities (docs/decisiones.md): a decimal
// string of the API becomes a bigint of ten-thousandths ("12500.5" →
// 125005000n), the precision of NUMERIC(18,4). A float never touches money.

const PLACES = 4;
const ONE = 10n ** BigInt(PLACES);
const CENT = 10n ** BigInt(PLACES - 2);
const decimalText = /^(-?)(\d+)(?:\.(\d{1,4}))?$/;

/** "12500.5" → 125005000n. Throws if the text is not a decimal of up to 4 decimals. */
export function toUnits(value: string): bigint {
  const match = decimalText.exec(value);
  if (!match) throw new Error(`not a decimal of up to ${PLACES} places: ${value}`);
  const [, sign, whole = '', fraction = ''] = match;
  const units = BigInt(whole) * ONE + BigInt(fraction.padEnd(PLACES, '0'));
  return sign ? -units : units;
}

/**
 * The API's format of an amount, as the backend writes it: 2 decimals, or
 * up to 4 when it has them (125005000n → "12500.50", 84033613n → "8403.3613").
 */
export function fromUnits(units: bigint): string {
  const sign = units < 0n ? '-' : '';
  const abs = units < 0n ? -units : units;
  let fraction = (abs % ONE).toString().padStart(PLACES, '0');
  while (fraction.length > 2 && fraction.endsWith('0')) fraction = fraction.slice(0, -1);
  return `${sign}${abs / ONE}.${fraction}`;
}

/** A quantity or a rate without trailing zeros: 35000n → "3.5", 20000n → "2". */
export function fromUnitsTrimmed(units: bigint): string {
  return fromUnits(units).replace(/\.?0+$/, '');
}

/** a × b rounded half-up (away from zero) to cents, like decimal's Round(2). */
export function mulRound2(a: bigint, b: bigint): bigint {
  return divide(a * b, ONE * CENT, 'halfUp') * CENT;
}

/** a × b truncated to cents (towards zero). */
export function mulTrunc2(a: bigint, b: bigint): bigint {
  return divide(a * b, ONE * CENT, 'truncate') * CENT;
}

function divide(n: bigint, d: bigint, mode: 'halfUp' | 'truncate'): bigint {
  const negative = n < 0n;
  const abs = negative ? -n : n;
  let q = abs / d;
  if (mode === 'halfUp' && (abs % d) * 2n >= d) q += 1n;
  return negative ? -q : q;
}
