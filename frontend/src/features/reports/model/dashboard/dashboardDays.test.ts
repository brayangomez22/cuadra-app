import { dashboardDays } from './dashboardDays';

const thursday = { lastWeek: '2026-11-12', weekStart: '2026-11-16', weeksFrom: '2026-09-28' };

describe('dashboardDays', () => {
  it.each([
    // a hardware store checks on a Thursday mid-morning
    {
      name: 'jueves a media mañana',
      now: '2026-11-19T15:30:00Z',
      want: { ...thursday, today: '2026-11-19', until: '10:30' },
    },
    // a clothing store closes on Thursday night: still Thursday in Colombia
    {
      name: 'jueves en la noche',
      now: '2026-11-20T04:45:00Z',
      want: { ...thursday, today: '2026-11-19', until: '23:45' },
    },
    {
      name: 'un lunes, la semana empieza ese día',
      now: '2026-11-16T14:05:00Z',
      want: {
        today: '2026-11-16',
        lastWeek: '2026-11-09',
        until: '09:05',
        weekStart: '2026-11-16',
        weeksFrom: '2026-09-28',
      },
    },
    {
      name: 'un domingo, la semana empezó el lunes',
      now: '2026-11-22T17:00:00Z',
      want: {
        today: '2026-11-22',
        lastWeek: '2026-11-15',
        until: '12:00',
        weekStart: '2026-11-16',
        weeksFrom: '2026-09-28',
      },
    },
    {
      name: 'a comienzo de año cruza al año anterior',
      now: '2027-01-04T13:00:00Z',
      want: {
        today: '2027-01-04',
        lastWeek: '2026-12-28',
        until: '08:00',
        weekStart: '2027-01-04',
        weeksFrom: '2026-11-16',
      },
    },
  ])('$name', ({ now, want }) => {
    expect(dashboardDays(new Date(now))).toEqual(want);
  });
});
