import { addDays, clockInBogota, formatDay, previousDay, todayInBogota, weekdayName, weekdayOf } from './dates';

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

describe('previousDay', () => {
  it('es el día anterior, también al cambiar de mes y de año', () => {
    expect(previousDay('2026-11-03')).toBe('2026-11-02');
    expect(previousDay('2026-03-01')).toBe('2026-02-28');
    expect(previousDay('2026-01-01')).toBe('2025-12-31');
  });
});

describe('addDays', () => {
  it('mueve un día hacia adelante o hacia atrás, también al cambiar de año', () => {
    expect(addDays('2026-11-19', -7)).toBe('2026-11-12');
    expect(addDays('2026-12-28', 7)).toBe('2027-01-04');
  });
});

describe('weekdayOf', () => {
  it('cuenta la semana desde el lunes', () => {
    expect(weekdayOf('2026-11-16')).toBe(0);
    expect(weekdayOf('2026-11-19')).toBe(3);
    expect(weekdayOf('2026-11-22')).toBe(6);
  });
});

describe('clockInBogota', () => {
  it('es la hora de Colombia con dos cifras, de 00:00 a 23:59', () => {
    expect(clockInBogota(new Date('2026-11-19T14:05:00Z'))).toBe('09:05');
    expect(clockInBogota(new Date('2026-11-20T04:59:00Z'))).toBe('23:59');
    expect(clockInBogota(new Date('2026-11-20T05:00:00Z'))).toBe('00:00');
  });
});

describe('weekdayName', () => {
  it('es el nombre del día en español, sin correrlo por la zona horaria', () => {
    expect(weekdayName('2026-11-19')).toBe('jueves');
    expect(weekdayName('2026-11-16')).toBe('lunes');
  });
});
