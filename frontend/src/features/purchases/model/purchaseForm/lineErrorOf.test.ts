import { ApiError } from '@/shared/api/errors';
import { lineErrorField } from './purchaseSchema';
import { lineErrorOf } from './lineErrorOf';

const apiError = (code: string, details?: Record<string, string>) =>
  new ApiError(422, { error: { code, message: 'Línea 3: …', details } });

describe('lineErrorOf', () => {
  it('lee la línea de details.line', () => {
    expect(lineErrorOf(apiError('invalid_quantity', { line: '2' }))).toBe(2);
    expect(lineErrorOf(apiError('invalid_quantity', { line: '0' }))).toBe(0);
  });

  it('un error sin línea, o que no es de la API, no tiene línea', () => {
    expect(lineErrorOf(apiError('purchase_invoice_taken'))).toBeUndefined();
    expect(lineErrorOf(apiError('invalid_quantity', { line: '1.5' }))).toBeUndefined();
    expect(lineErrorOf(new Error('sin conexión'))).toBeUndefined();
  });
});

describe('lineErrorField', () => {
  it.each([
    ['quantity_precision', 1, 'lines.1.quantity'],
    ['unit_not_available', 0, 'lines.0.unit_id'],
    ['invalid_unit_cost', 3, 'lines.3.unit_cost'],
    ['invalid_tax_rate', 2, 'lines.2.tax_rate'],
    ['variant_inactive', 0, 'lines.0.item'],
  ])('%s de la línea %i va a %s', (code, line, field) => {
    expect(lineErrorField(code, line)).toBe(field);
  });
});
