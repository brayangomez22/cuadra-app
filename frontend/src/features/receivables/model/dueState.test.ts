import { dueState } from './dueState';

describe('dueState', () => {
  it.each([
    ['2026-10-27', { state: 'overdue', days: 1 }],
    ['2026-09-28', { state: 'overdue', days: 30 }],
    ['2026-10-28', { state: 'due_today', days: 0 }],
    ['2026-11-27', { state: 'upcoming', days: 30 }],
  ])('una cuenta que vence el %s, vista el 28/10/2026', (dueDate, expected) => {
    expect(dueState(dueDate, '2026-10-28')).toEqual(expected);
  });

  it('cuenta los días aunque haya un cambio de mes o de año', () => {
    expect(dueState('2026-12-31', '2027-01-02')).toEqual({ state: 'overdue', days: 2 });
  });
});
