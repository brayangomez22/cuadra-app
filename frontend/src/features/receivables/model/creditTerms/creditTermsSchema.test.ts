import { creditTermsDefaults, creditTermsSchema } from './creditTermsSchema';

describe('creditTermsSchema', () => {
  // A builder of a hardware store and a boutique of a clothing store.
  it.each([
    ['1.500.000', '30', { credit_limit: '1500000', term_days: 30 }],
    ['800000,50', '0', { credit_limit: '800000.50', term_days: 0 }],
    ['0', '365', { credit_limit: '0', term_days: 365 }],
  ])('lee el cupo «%s» y el plazo «%s»', (credit_limit, term_days, expected) => {
    expect(creditTermsSchema.parse({ credit_limit, term_days })).toEqual(expected);
  });

  it.each([
    [{ credit_limit: '-1', term_days: '30' }, 'credit_limit', 'El valor no puede ser negativo'],
    [{ credit_limit: '', term_days: '30' }, 'credit_limit', 'Escriba el cupo, aunque sea 0'],
    [{ credit_limit: '100000', term_days: '366' }, 'term_days', 'El plazo va de 0 a 365 días'],
    [{ credit_limit: '100000', term_days: '15,5' }, 'term_days', 'Escriba el plazo en días enteros'],
    [{ credit_limit: '100000', term_days: '' }, 'term_days', 'Escriba el plazo en días enteros'],
  ])('rechaza %o', (values, field, message) => {
    const issue = creditTermsSchema.safeParse(values).error?.issues[0];
    expect(issue?.path).toEqual([field]);
    expect(issue?.message).toBe(message);
  });
});

describe('creditTermsDefaults', () => {
  it('escribe el cupo y el plazo como los teclea el usuario', () => {
    expect(creditTermsDefaults({ credit_limit: '1500000.00', term_days: 30 })).toEqual({
      credit_limit: '1500000',
      term_days: '30',
    });
    expect(creditTermsDefaults({ credit_limit: '800000.50', term_days: 0 })).toEqual({
      credit_limit: '800000,50',
      term_days: '0',
    });
  });
});
