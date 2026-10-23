import { cleanup, screen, within } from '@testing-library/react';
import { apiError, json } from '@/test/fakeApi';
import { aLocation, anOpenCashSession, ids } from '@/test/fixtures';
import { posApi } from '../../testing/posApi';
import { checkout, clearPosStorage, findCartLine, posted, scan, user } from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('sede', () => {
    it('con varias sedes, pide elegir una antes de cobrar y la recuerda', async () => {
      posApi({ locations: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();
      expect(screen.getByText('Elija la sede para cobrar.')).toBeInTheDocument();

      await user().click(screen.getByRole('combobox', { name: 'Sede' }));
      await user().click(await screen.findByRole('option', { name: 'Sede norte' }));
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeEnabled();

      cleanup();
      renderApp('/vender');
      expect(await screen.findByRole('combobox', { name: 'Sede' })).toHaveTextContent('Sede norte');
    });

    it('sin sedes activas explica qué hacer', async () => {
      posApi({ locations: [aLocation({ active: false })] });
      renderApp('/vender');

      expect(await screen.findByText('No hay sedes activas para vender')).toBeInTheDocument();
    });
  });

  describe('caja', () => {
    it('si la empresa exige caja y no hay una abierta, no deja vender y lleva a abrirla', async () => {
      posApi({ cash: { required: true } });
      renderApp('/vender');

      expect(await screen.findByRole('heading', { name: 'Abra la caja para vender' })).toBeInTheDocument();
      expect(screen.getByRole('link', { name: 'Abrir caja' })).toHaveAttribute('href', '/caja');
      expect(screen.queryByRole('combobox', { name: 'Buscar producto' })).not.toBeInTheDocument();
    });

    it('con la caja abierta vende en la sede de la caja, aunque haya varias', async () => {
      const api = posApi({
        cash: { required: true, session: anOpenCashSession({ location_id: ids.north }) },
        locations: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })],
      });
      renderApp('/vender');

      await scan('CEM-50');
      await findCartLine(/Cemento/);
      expect(screen.queryByRole('combobox', { name: 'Sede' })).not.toBeInTheDocument();
      expect(screen.getByText(/Caja abierta en Sede norte/)).toBeInTheDocument();
      await user().keyboard('{F4}');
      await user().keyboard('{Enter}');

      expect((await posted(api)).location_id).toBe(ids.north);
    });

    it('si la caja se cerró mientras vendía, lo dice y vuelve a pedirla', async () => {
      let current: unknown = { required: true, session: anOpenCashSession() };
      posApi({
        extra: {
          'GET /api/v1/cash-sessions/current': () => json(200, current),
          'POST /api/v1/sales': () => {
            current = { required: true };
            return apiError(409, {
              code: 'cash_session_closed',
              message: 'La caja ya está cerrada. Para seguir vendiendo, abra una nueva.',
            });
          },
        },
      });
      renderApp('/vender');

      await scan('CEM-50');
      await findCartLine(/Cemento/);
      await user().keyboard('{F4}');
      await user().keyboard('{Enter}');

      expect(await within(await checkout()).findByText(/La caja ya está cerrada/)).toBeInTheDocument();
      await user().keyboard('{Escape}');
      expect(await screen.findByRole('heading', { name: 'Abra la caja para vender' })).toBeInTheDocument();
    });
  });
});
