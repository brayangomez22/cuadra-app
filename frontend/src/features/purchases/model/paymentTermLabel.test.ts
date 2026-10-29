import { paymentTermLabel } from './paymentTermLabel';

describe('paymentTermLabel', () => {
  it.each([
    [0, 'Contado'],
    [1, '1 día'],
    [30, '30 días'],
  ])('%i días de plazo se muestran como «%s»', (days, label) => {
    expect(paymentTermLabel(days)).toBe(label);
  });
});
