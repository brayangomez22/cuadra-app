import { screen, within } from '@testing-library/react';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { ids, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { anOpenSummary } from '../../testing/cashHistory';
import { cashApi } from '../../testing/cashApi';
import { aClosedCashSession, anOpenCashSession } from '../../testing/fixtures';
import { clearPosStorage } from '../../testing/posSteps';

beforeEach(clearPosStorage);

const sessionKey = `GET /api/v1/cash-sessions/${ids.cashSession}` as const;

describe('CashReportPage: total vendido y cajas de otros', () => {
  it('muestra el total vendido de la caja', async () => {
    cashApi({ [sessionKey]: json(200, aClosedCashSession()) });
    renderApp(`/caja/${ids.cashSession}`);

    const report = within(await screen.findByRole('article', { name: 'Cierre de caja' }));
    expect(plain(report.getByText('Total vendido').nextElementSibling?.textContent)).toBe('$ 105.000');
  });

  it('dice cuándo la cerró alguien distinto del cajero', async () => {
    const closed = aClosedCashSession();
    cashApi({
      [sessionKey]: json(200, {
        ...closed,
        closing: { ...closed.closing!, closed_by: '01920000-0000-7000-8000-000000000d07' },
      }),
    });
    renderApp(`/caja/${ids.cashSession}`);

    const report = within(await screen.findByRole('article', { name: 'Cierre de caja' }));
    expect(report.getByText('Cerrada por').nextElementSibling).toHaveTextContent('El dueño o un administrador');
  });

  it('a quien maneja la caja, una caja ajena abierta le dice de quién es y le ofrece cerrarla', async () => {
    const userId = '01920000-0000-7000-8000-000000000d09';
    cashApi({
      ...signedIn(aUser('admin')),
      [sessionKey]: json(200, anOpenCashSession({ user_id: userId })),
      'GET /api/v1/cash-sessions': json(200, {
        items: [anOpenSummary({ id: ids.cashSession, location_id: ids.main, user_id: userId })],
        total: 1,
      }),
    });
    renderApp(`/caja/${ids.cashSession}`);

    expect(await screen.findByRole('link', { name: 'Cerrar la caja de Pedro Ruiz' })).toHaveAttribute(
      'href',
      `/caja/${ids.cashSession}/cerrar`,
    );
    expect(
      screen.getByText(/La abrió Pedro Ruiz en Bodega principal el 13\/10\/2026, 8:00 a\. m\./),
    ).toBeInTheDocument();
  });
});
