import { taxLabel } from './taxRates';

describe('taxLabel', () => {
  it.each([
    { negocio: 'ferretería', tax_rate: '0.19', tax_treatment: 'taxed' as const, want: 'IVA 19 %' },
    { negocio: 'papelería', tax_rate: '0', tax_treatment: 'exempt' as const, want: 'IVA exento' },
    { negocio: 'droguería', tax_rate: '0', tax_treatment: 'excluded' as const, want: 'Sin IVA (excluido)' },
  ])('el IVA de un producto de $negocio es «$want»', ({ want, ...product }) => {
    expect(taxLabel(product).replace(/\s/g, ' ')).toBe(want);
  });
});
