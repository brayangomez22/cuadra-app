import { ids } from '@/test/fixtures';
import { anOpenSummary } from '../../testing/cashHistory';
import { anOpenCashSession } from '../../testing/fixtures';
import { drawerOwner, openedBy, whereAndSince } from './drawerOwner';

const session = anOpenCashSession();
const locations = [{ id: ids.main, name: 'Bodega principal' }];

describe('drawerOwner', () => {
  it('toma el nombre del cajero de las cajas abiertas y la sede de las sedes', () => {
    const owner = drawerOwner(session, { open: [anOpenSummary({ id: session.id })], locations });
    expect(owner).toEqual({ cashier: 'Pedro Ruiz', location: 'Bodega principal', openedAt: '13/10/2026, 8:00 a. m.' });
    expect(whereAndSince(owner).replace(/\s/g, ' ')).toBe('Bodega principal · abierta el 13/10/2026, 8:00 a. m.');
    expect(openedBy(owner).replace(/\s/g, ' ')).toBe(
      'La abrió Pedro Ruiz en Bodega principal el 13/10/2026, 8:00 a. m. ',
    );
  });

  it('sin el nombre no dice quién la abrió, pero sí dónde y desde cuándo', () => {
    const owner = drawerOwner(session, { open: [], locations: [] });
    expect(owner.cashier).toBeUndefined();
    expect(openedBy(owner)).toBe('');
    expect(whereAndSince(owner).replace(/\s/g, ' ')).toBe('abierta el 13/10/2026, 8:00 a. m.');
  });
});
