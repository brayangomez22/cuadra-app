import { aLocation, ids } from '@/test/fixtures';
import { locationOptions, shownLocation } from './shownLocation';

describe('shownLocation', () => {
  const closed = aLocation({ id: '01920000-0000-7000-8000-000000000299', name: 'Bodega cerrada', active: false });
  const north = aLocation({ id: ids.north, name: 'Sede norte' });

  it('usa la sede de la URL si existe', () => {
    expect(shownLocation([closed, north], closed.id)).toBe(closed);
  });

  it('sin sede válida elige la primera activa', () => {
    expect(shownLocation([closed, north], 'otra')).toBe(north);
    expect(shownLocation([closed, north], '')).toBe(north);
  });

  it('si todas están inactivas elige la primera', () => {
    expect(shownLocation([closed], '')).toBe(closed);
  });

  it('sin sedes no hay sede', () => {
    expect(shownLocation([], '')).toBeUndefined();
  });
});

describe('locationOptions', () => {
  it('marca las sedes inactivas', () => {
    expect(locationOptions([aLocation(), aLocation({ id: ids.north, name: 'Sede norte', active: false })])).toEqual([
      { value: ids.main, label: 'Bodega principal' },
      { value: ids.north, label: 'Sede norte (inactiva)' },
    ]);
  });
});
