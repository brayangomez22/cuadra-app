// Formats for amounts, quantities and dates as Colombians read them. Amounts and
// quantities travel as decimal strings ("12500.50") and never become a float:
// Intl.NumberFormat formats a numeric string with exact decimal precision.

type NumericString = Intl.StringNumericLiteral;

const locale = 'es-CO';
const timeZone = 'America/Bogota';

const moneyWhole = new Intl.NumberFormat(locale, {
  style: 'currency',
  currency: 'COP',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});
const moneyCents = new Intl.NumberFormat(locale, {
  style: 'currency',
  currency: 'COP',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const quantity = new Intl.NumberFormat(locale, { maximumFractionDigits: 4 });
const signedQuantity = new Intl.NumberFormat(locale, { maximumFractionDigits: 4, signDisplay: 'exceptZero' });
const rate = new Intl.NumberFormat(locale, { style: 'percent', maximumFractionDigits: 2 });
const percent = new Intl.NumberFormat(locale, { maximumFractionDigits: 2 });
const dateTime = new Intl.DateTimeFormat(locale, {
  timeZone,
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: 'numeric',
  minute: '2-digit',
});

const hasCents = (value: string) => /\.\d*[1-9]/.test(value);

/** "12500.00" → "$ 12.500"; "12500.5" → "$ 12.500,50" (half-up to 2 decimals). */
export function formatMoney(value: string): string {
  return (hasCents(value) ? moneyCents : moneyWhole).format(value as NumericString);
}

/** "2.5" → "2,5"; with signed, an entry shows its "+". */
export function formatQuantity(value: string, { signed = false } = {}): string {
  return (signed ? signedQuantity : quantity).format(value as NumericString);
}

/**
 * A tax rate as a fraction ("0.19", kind rate) or a value already in percent
 * ("16.6667", kind percent) → "19 %", "16,67 %".
 */
export function formatPercent(value: string, kind: 'rate' | 'percent'): string {
  const number =
    kind === 'rate'
      ? rate
          .formatToParts(value as NumericString)
          .filter((part) => part.type !== 'percentSign' && part.type !== 'literal')
          .map((part) => part.value)
          .join('')
      : percent.format(value as NumericString);
  return `${number}\u00a0%`;
}

/** An instant of the API in Bogotá time: "10/10/2026, 3:05 p. m.". */
export function formatDateTime(iso: string): string {
  return dateTime.format(new Date(iso));
}

export type DecimalInput =
  { ok: true; value: string } | { ok: false; reason: 'invalid' | 'thousands' | 'decimals' | 'too_large' };

// "12.500" or "1.250.000": a dot followed by groups of three digits reads as a
// thousands separator in Colombia, so it is ambiguous with a decimal dot.
const thousandsLike = /^-?\d{1,3}(\.\d{3})+$/;
const decimalText = /^(-?)(\d+)(?:[.,](\d+))?$/;

/**
 * Reads what a person typed in an amount or quantity field. The decimal
 * separator is a comma or a dot; thousands separators are not accepted. The
 * result is the API's format: "12500,50" → "12500.50".
 */
export function parseDecimalInput(text: string): DecimalInput {
  const trimmed = text.trim();
  if (thousandsLike.test(trimmed)) return { ok: false, reason: 'thousands' };
  const match = decimalText.exec(trimmed);
  if (!match) return { ok: false, reason: 'invalid' };
  const [, sign = '', whole = '', fraction] = match;
  if (fraction !== undefined && fraction.length > 4) return { ok: false, reason: 'decimals' };
  if (whole.replace(/^0+(?=\d)/, '').length > 14) return { ok: false, reason: 'too_large' };
  return { ok: true, value: fraction === undefined ? `${sign}${whole}` : `${sign}${whole}.${fraction}` };
}

// "100.000" or "1.250.000,50": an amount with Colombian thousands separators.
const moneyWithThousands = /^\d{1,3}(?:\.\d{3})+(?:,\d{1,2})?$/;

/**
 * Reads an amount of money as a cashier types it. Unlike a quantity, an
 * amount in pesos is never 1,5 thousandths, so "100.000" is a hundred
 * thousand: dots in groups of three are thousands separators.
 */
export function parseMoneyInput(text: string): DecimalInput {
  const trimmed = text.trim();
  if (moneyWithThousands.test(trimmed)) return parseDecimalInput(trimmed.replaceAll('.', ''));
  return parseDecimalInput(trimmed);
}

/** A decimal string of the API as it is typed in a field: "10000.00" → "10000,00". */
export function toDecimalInput(value: string): string {
  return value.replace('.', ',');
}

/** Whether a decimal string of the API is zero ("0", "0.00", "-0"). */
export function isZero(value: string): boolean {
  return /^-?0+(\.0+)?$/.test(value);
}

/** Whether a decimal string of the API is below zero. */
export function isNegative(value: string): boolean {
  return value.startsWith('-') && !isZero(value);
}
