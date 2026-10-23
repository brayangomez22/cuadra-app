import { aGarment, aProduct, variantBarcode } from '@/test/fixtures';
import { isInactiveCode, readSearch, searchOutcome } from './searchText';

describe('readSearch', () => {
  it.each([
    ['3*7701234567890', { quantity: '3', term: '7701234567890' }],
    ['2,5*cinta', { quantity: '2.5', term: 'cinta' }],
    [' 2 * CAM-M-ROJO ', { quantity: '2', term: 'CAM-M-ROJO' }],
  ])('lee la cantidad antes del código en «%s»', (text, expected) => {
    expect(readSearch(text)).toEqual(expected);
  });

  it.each([
    ['tubo pvc', 'tubo pvc'],
    ['0*CEM-50', '0*CEM-50'],
  ])('sin una cantidad válida, «%s» se busca con cantidad 1', (text, term) => {
    expect(readSearch(text)).toEqual({ quantity: '1', term });
  });
});

describe('isInactiveCode', () => {
  it('reconoce el código de barras o el SKU de una variante que no se ofrece', () => {
    const products = [aProduct(), aGarment()];
    expect(isInactiveCode(products, variantBarcode('M', 'Rojo'))).toBe(true);
    expect(isInactiveCode(products, 'tub-pvc-12')).toBe(true);
  });

  it('un nombre no es un código', () => {
    expect(isInactiveCode([aProduct(), aGarment()], 'Camiseta')).toBe(false);
  });
});

describe('searchOutcome', () => {
  const inactiveRed = () => {
    const garment = aGarment();
    return {
      ...garment,
      variants: garment.variants.map((v) => (v.label === 'Rojo · M' ? { ...v, active: false } : v)),
    };
  };

  it('el código exacto de una variante la agrega aunque haya más resultados', () => {
    const outcome = searchOutcome([aGarment()], variantBarcode('M', 'Rojo'));
    expect(outcome.type === 'add' && outcome.sellable.variant.label).toBe('Rojo · M');
  });

  it('un único resultado se agrega', () => {
    const outcome = searchOutcome([aProduct()], 'tubo');
    expect(outcome.type === 'add' && outcome.sellable.variant.sku).toBe('TUB-PVC-12');
  });

  it('varios resultados se eligen con las flechas', () => {
    expect(searchOutcome([aGarment()], 'camiseta')).toEqual({ type: 'choose' });
  });

  it('sin resultados, o con el código de una variante inactiva, no lo encuentra', () => {
    expect(searchOutcome([], 'CEM-50')).toEqual({ type: 'not_found' });
    expect(searchOutcome([inactiveRed()], variantBarcode('M', 'Rojo'))).toEqual({ type: 'not_found' });
  });
});
