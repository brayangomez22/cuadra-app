import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { generalApi } from '../../testing/settingsApi';

describe('GeneralSettings: reglas del negocio', () => {
  it('muestra y guarda el descuento máximo del cajero y la caja obligatoria', async () => {
    // Almacén de ropa: sus cajeros descuentan hasta 15 % en temporada y venden sin caja.
    const api = generalApi({
      'PATCH /api/v1/sales/settings': json(200, { max_discount: '0.15', require_cash_session: false }),
    });
    renderApp('/configuracion/general');

    const discount = await screen.findByLabelText('Descuento máximo del cajero (%)');
    expect(discount).toHaveValue('0');
    const cash = screen.getByRole('checkbox', { name: 'Exigir una caja abierta para vender' });
    expect(cash).toBeChecked();
    await userEvent.clear(discount);
    await userEvent.type(discount, '15');
    await userEvent.click(cash);
    await userEvent.click(screen.getByRole('button', { name: 'Guardar reglas de venta' }));

    expect(await screen.findByText('Reglas de venta guardadas')).toBeInTheDocument();
    expect(await api.calls('PATCH /api/v1/sales/settings')[0]!.json()).toEqual({
      max_discount: '0.15',
      require_cash_session: false,
    });
  });

  it('solo el bloque con cambios sin guardar destaca su botón', async () => {
    generalApi();
    renderApp('/configuracion/general');

    const discount = await screen.findByLabelText('Descuento máximo del cajero (%)');
    const salesButton = screen.getByRole('button', { name: 'Guardar reglas de venta' });
    expect(salesButton).toHaveAttribute('data-variant', 'secondary');
    expect(screen.getByRole('button', { name: 'Guardar datos del negocio' })).toHaveAttribute(
      'data-variant',
      'secondary',
    );
    await userEvent.type(discount, '5');

    expect(salesButton).toHaveAttribute('data-variant', 'primary');
    expect(screen.getByRole('button', { name: 'Guardar datos del negocio' })).toHaveAttribute(
      'data-variant',
      'secondary',
    );
  });

  it('no envía un descuento de más de 100 %', async () => {
    const api = generalApi();
    renderApp('/configuracion/general');

    const discount = await screen.findByLabelText('Descuento máximo del cajero (%)');
    await userEvent.clear(discount);
    await userEvent.type(discount, '120');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar reglas de venta' }));

    expect(await screen.findByText('El descuento va de 0 a 100 %')).toBeInTheDocument();
    expect(api.calls('PATCH /api/v1/sales/settings')).toHaveLength(0);
  });

  it('permite el stock negativo', async () => {
    // Ferretería que vende el cemento antes de registrar la compra.
    const api = generalApi({ 'PATCH /api/v1/inventory/settings': json(200, { allow_negative_stock: true }) });
    renderApp('/configuracion/general');

    const negative = await screen.findByRole('checkbox', { name: 'Permitir vender sin existencias (stock negativo)' });
    expect(negative).not.toBeChecked();
    await userEvent.click(negative);
    await userEvent.click(screen.getByRole('button', { name: 'Guardar reglas de inventario' }));

    expect(await screen.findByText('Reglas de inventario guardadas')).toBeInTheDocument();
    expect(await api.calls('PATCH /api/v1/inventory/settings')[0]!.json()).toEqual({ allow_negative_stock: true });
  });

  it('si no cargan las reglas lo dice y deja guardar los datos del negocio', async () => {
    generalApi({
      'GET /api/v1/sales/settings': json(500, { error: { code: 'internal', message: 'Algo falló.' } }),
    });
    renderApp('/configuracion/general');

    expect(await screen.findByText('No pudimos cargar las reglas del negocio')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Guardar datos del negocio' })).toBeEnabled();
  });
});
