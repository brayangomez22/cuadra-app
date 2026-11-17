// The catalog's document, as text: a test checks they agree.
import doc from '../../../../../docs/eventos.md?raw';
import { allowedProps } from './allowedProps';
import { eventCatalog, type EventName } from './catalog';

describe('allowedProps', () => {
  it('ningún evento lleva campos fuera de la lista permitida del catálogo', () => {
    // Lo que una pantalla podría pasar por error: datos de un cliente de una
    // ferretería o de un almacén de calzado.
    const leaked = {
      name: 'Ana Gómez',
      email: 'ana@eltornillo.co',
      document_number: '1020304050',
      query: 'botas talla 38',
      total: '125000.00',
      customer: { name: 'Calzado Andino' },
    };
    for (const event of Object.keys(eventCatalog) as EventName[]) {
      const sent = allowedProps(event, leaked);
      expect(Object.keys(sent)).toEqual(['catalog_version']);
    }
  });

  it('deja pasar las propiedades del catálogo', () => {
    expect(allowedProps('sale_confirmed', { lines: 3, payment_methods: 2, with_customer: true })).toEqual({
      catalog_version: 1,
      lines: 3,
      payment_methods: 2,
      with_customer: true,
    });
    expect(allowedProps('signup_completed', { business_template: 'ropa_calzado' })).toEqual({
      catalog_version: 1,
      business_template: 'ropa_calzado',
    });
  });

  it('descarta un texto largo aunque su llave esté permitida', () => {
    expect(allowedProps('error_shown', { code: 'x'.repeat(65), screen: '/vender' })).toEqual({
      catalog_version: 1,
      screen: '/vender',
    });
  });

  it('cada evento del catálogo está documentado en docs/eventos.md', () => {
    for (const [event, props] of Object.entries(eventCatalog)) {
      expect(doc).toContain(`\`${event}\``);
      for (const prop of props) expect(doc).toContain(`\`${prop}\``);
    }
  });
});
