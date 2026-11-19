import { screen, within } from '@testing-library/react';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { dashboardNow, teamWeeks } from '../../testing/dashboardFixtures';
import { cashier } from '../../testing/teamFixtures';

const teamKey = 'GET /api/v1/reports/team-control';

type Routes = Parameters<typeof fakeApi>[0];

function teamApi(routes: Routes = {}, user = aUser()) {
  return fakeApi({ ...signedIn(user), [teamKey]: json(200, teamWeeks), ...routes });
}

const card = async () => within(await screen.findByRole('region', { name: 'Control del equipo esta semana' }));

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(dashboardNow);
});
afterEach(() => vi.useRealTimers());

describe('TeamAlerts en el inicio', () => {
  it('alerta lo de esta semana que pasa del doble de lo habitual de esa persona, en las últimas 8 semanas', async () => {
    const api = teamApi();
    renderApp('/');

    const alerts = await card();
    const [luis, ...others] = await alerts.findAllByRole('listitem');
    expect(others).toHaveLength(0);
    expect(plain(luis?.textContent)).toContain('Luis Pérez');
    expect(plain(luis?.textContent)).toContain('5 anulaciones');
    expect(plain(luis?.textContent)).toContain('$ 8.000 de faltantes de caja');
    expect(plain(luis?.textContent)).not.toContain('descuentos');
    const [request] = api.calls(teamKey);
    expect([query(request!).get('from'), query(request!).get('to'), query(request!).get('bucket')]).toEqual([
      '2026-09-28',
      '2026-11-19',
      'week',
    ]);
  });

  it('cada alerta enlaza a la tendencia del reporte, donde esa semana sale resaltada, y a la auditoría', async () => {
    teamApi();
    renderApp('/');

    const alerts = await card();
    expect(await alerts.findByRole('link', { name: 'Ver la tendencia de anulaciones de Luis Pérez' })).toHaveAttribute(
      'href',
      '/reportes/equipo?desde=2026-09-28&hasta=2026-11-19&cada=week&cifra=voids',
    );
    expect(alerts.getByRole('link', { name: 'Ver lo que hizo Luis Pérez esta semana' })).toHaveAttribute(
      'href',
      `/auditoria?usuario=${cashier.user_id}&desde=2026-11-16&hasta=2026-11-19`,
    );
  });

  it('el administrador, que no ve la auditoría, no recibe ese enlace', async () => {
    teamApi({}, aUser('admin'));
    renderApp('/');

    const alerts = await card();
    expect(await alerts.findByRole('link', { name: /Ver la tendencia de anulaciones/ })).toBeInTheDocument();
    expect(alerts.queryByRole('link', { name: /Ver lo que hizo/ })).not.toBeInTheDocument();
  });

  it('sin nada fuera de lo normal lo dice', async () => {
    teamApi({ [teamKey]: json(200, { ...teamWeeks, users: [], series: [] }) });
    renderApp('/');

    expect(await (await card()).findByText('Nada fuera de lo normal esta semana.')).toBeInTheDocument();
  });

  it('si falla muestra el error dentro de la tarjeta', async () => {
    teamApi({ [teamKey]: apiError(500, { code: 'internal', message: 'Falló' }) });
    renderApp('/');

    expect(await (await card()).findByText('No pudimos revisar el control del equipo')).toBeInTheDocument();
  });
});
