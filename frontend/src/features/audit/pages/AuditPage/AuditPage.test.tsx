import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { auditApi, cashierUser, saleId } from '../../testing/auditApi';

const auditKey = 'GET /api/v1/audit-events';

describe('AuditPage', () => {
  it('muestra quién hizo qué y cuándo, con el detalle y el enlace a la venta', async () => {
    auditApi();
    renderApp('/auditoria');

    const row = within((await screen.findByRole('cell', { name: /^Anuló una venta/ })).closest('tr')!);
    expect(row.getByText('Luis Pérez')).toBeInTheDocument();
    expect(row.getByText('Venta de $ 185.000')).toBeInTheDocument();
    expect(row.getByRole('link', { name: 'Ver venta' })).toHaveAttribute('href', `/ventas/${saleId}`);
    expect(screen.getByText('De $ 4.500 a $ 4.800')).toBeInTheDocument();
  });

  it('filtra por acción y por usuario en la URL y en la consulta', async () => {
    const api = auditApi();
    renderApp('/auditoria');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Acción' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Anuló una venta' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'Usuario' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Luis Pérez' }));

    await waitFor(() => {
      const last = new URL(api.calls(auditKey).at(-1)!.url);
      expect(last.searchParams.get('action')).toBe('sale.voided');
      expect(last.searchParams.get('user_id')).toBe(cashierUser.id);
    });
  });

  it('el administrador no entra: la auditoría es del dueño', async () => {
    auditApi({}, aUser('admin'));
    renderApp('/auditoria');

    expect(await screen.findByText(/No tiene permiso/)).toBeInTheDocument();
  });
});
