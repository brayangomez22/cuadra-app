import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { ids } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { CLOSE_KEY, cashApi } from '../../testing/cashApi';
import { aClosedCashSession, anOpenCashSession } from '../../testing/fixtures';

const othersDrawer = anOpenCashSession({ user_id: '01920000-0000-7000-8000-000000000d09' });

describe('CloseCashOfPage', () => {
  it('el administrador cuenta la caja que otro dejó abierta y ve el cierre', async () => {
    let session = othersDrawer;
    const api = cashApi({
      ...signedIn(aUser('admin')),
      [`GET /api/v1/cash-sessions/${ids.cashSession}`]: () => json(200, session),
      [CLOSE_KEY]: () => {
        session = aClosedCashSession({ user_id: othersDrawer.user_id });
        return json(200, session);
      },
    });
    renderApp(`/caja/${ids.cashSession}/cerrar`);

    expect(await screen.findByRole('heading', { name: 'Cerrar la caja de otro usuario' })).toBeInTheDocument();
    await userEvent.type(await screen.findByLabelText('Billetes de $ 100.000'), '1');
    await userEvent.click(screen.getByRole('button', { name: 'Cerrar caja' }));
    const confirm = await screen.findByRole('dialog', { name: '¿Cerrar la caja?' });
    await userEvent.click(within(confirm).getByRole('button', { name: 'Cerrar caja' }));

    expect(await screen.findByRole('heading', { name: 'Cierre de caja' })).toBeInTheDocument();
    expect(api.calls(CLOSE_KEY)).toHaveLength(1);
  });

  it('una caja ya cerrada lleva a su cierre', async () => {
    cashApi({
      ...signedIn(aUser('admin')),
      [`GET /api/v1/cash-sessions/${ids.cashSession}`]: json(200, aClosedCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}/cerrar`);

    expect(await screen.findByRole('heading', { name: 'Cierre de caja' })).toBeInTheDocument();
  });

  it('el cajero no cierra cajas ajenas', async () => {
    cashApi({ [`GET /api/v1/cash-sessions/${ids.cashSession}`]: json(200, othersDrawer) });
    renderApp(`/caja/${ids.cashSession}/cerrar`);

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
  });
});
