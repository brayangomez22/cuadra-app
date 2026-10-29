import { formatDay, todayInBogota } from './dates';

describe('formatDay', () => {
  it('muestra un día de la API en formato dd/mm/aaaa, sin correrlo por la zona horaria', () => {
    expect(formatDay('2026-11-05')).toBe('05/11/2026');
    expect(formatDay('2027-01-01')).toBe('01/01/2027');
  });
});

describe('todayInBogota', () => {
  it('es el día en hora de Colombia, aunque en UTC ya sea el siguiente', () => {
    expect(todayInBogota(new Date('2026-10-29T03:30:00Z'))).toBe('2026-10-28');
    expect(todayInBogota(new Date('2026-10-29T05:00:00Z'))).toBe('2026-10-29');
  });
});
