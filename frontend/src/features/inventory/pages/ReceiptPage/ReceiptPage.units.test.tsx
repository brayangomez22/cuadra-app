import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, json, type Me } from '@/test/fakeApi';
import { aProduct, boxOf12, ids, page, unitOf, withUnits } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aMovement } from '../../testing/fixtures';
import { receiptApi } from '../../testing/receiptApi';
import { pickLocation, pickProduct, register } from '../../testing/receiptSteps';

describe('ReceiptPage', () => {
  describe('en unidad alternativa', () => {
    const withUnitConversions = (): Me => ({ ...aUser(), modules: ['unit_conversions'] });
    const tubes = withUnits(aProduct(), [boxOf12]);
    const products = { 'GET /api/v1/products': json(200, page([tubes])) };

    it('registra una entrada en cajas: envía unit_id y avisa lo que entró en la unidad base', async () => {
      const api = receiptApi(
        {
          ...products,
          'POST /api/v1/inventory/receipts': json(201, aMovement({ quantity: '24', balance_after: '144' })),
        },
        withUnitConversions(),
      );
      renderApp('/inventario/entrada');

      await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
      await pickLocation('Bodega principal');
      await userEvent.click(screen.getByRole('combobox', { name: 'Unidad' }));
      await userEvent.click(await screen.findByRole('option', { name: 'caja (12 und)' }));
      expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(
        expect.stringContaining('En caja de 12 und.'),
      );
      // The suggested cost is the base unit's times the factor: 8.333,33 × 12.
      expect(screen.getByLabelText('Costo unitario')).toHaveValue('99999,96');
      await userEvent.type(screen.getByLabelText('Cantidad'), '2');
      await userEvent.clear(screen.getByLabelText('Costo unitario'));
      await userEvent.type(screen.getByLabelText('Costo unitario'), '96000');
      await register();

      expect(await screen.findByText('Entrada registrada')).toBeInTheDocument();
      expect(screen.getByText(/^2 caja \(24 und\) de Tubo PVC/)).toBeInTheDocument();
      expect(await api.calls('POST /api/v1/inventory/receipts')[0]?.json()).toEqual({
        variant_id: ids.pipe,
        location_id: ids.main,
        quantity: '2',
        unit_cost: '96000',
        unit_id: unitOf('caja').id,
      });
    });

    it('en la unidad base no envía unit_id', async () => {
      const api = receiptApi(products, withUnitConversions());
      renderApp('/inventario/entrada');

      await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
      await pickLocation('Bodega principal');
      expect(screen.getByRole('combobox', { name: 'Unidad' })).toHaveTextContent('und');
      await userEvent.type(screen.getByLabelText('Cantidad'), '10');
      await register();

      await screen.findByText('Entrada registrada');
      expect(await api.calls('POST /api/v1/inventory/receipts')[0]?.json()).not.toHaveProperty('unit_id');
    });

    it('sin el módulo no aparece el selector de unidad', async () => {
      receiptApi(products);
      renderApp('/inventario/entrada');

      await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');

      expect(screen.queryByRole('combobox', { name: 'Unidad' })).not.toBeInTheDocument();
    });
  });
});
