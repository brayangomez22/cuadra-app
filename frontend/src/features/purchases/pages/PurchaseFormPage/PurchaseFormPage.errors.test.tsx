import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { purchasesApi } from '../../testing/purchasesApi';
import { fillAmounts, fillInvoice, lineGroup, pickProduct, registerPurchase } from '../../testing/purchaseSteps';

async function fillOneLine() {
  await fillInvoice();
  await pickProduct(1, 'cemento', 'Cemento gris 50kg');
  await fillAmounts(1, '10', '28000');
}

async function fillTwoLines() {
  await fillOneLine();
  await userEvent.click(screen.getByRole('button', { name: 'Agregar línea' }));
  await pickProduct(2, 'tubo', 'Tubo PVC');
  await fillAmounts(2, '2,5', '8000');
}

describe('PurchaseFormPage: errores', () => {
  it('la factura repetida se marca en el número', async () => {
    purchasesApi({
      routes: {
        'POST /api/v1/purchases': apiError(409, {
          code: 'purchase_invoice_taken',
          message: 'La factura FE 1234 de este proveedor ya está registrada.',
        }),
      },
    });
    renderApp('/compras/nueva');

    await fillOneLine();
    await registerPurchase();

    const number = screen.getByLabelText('Número de factura');
    expect(await screen.findByText('La factura FE 1234 de este proveedor ya está registrada.')).toBeInTheDocument();
    expect(number).toHaveAttribute('aria-invalid', 'true');
    expect(number).toHaveFocus();
  });

  it('el error de una línea queda en su fila', async () => {
    purchasesApi({
      routes: {
        'POST /api/v1/purchases': apiError(422, {
          code: 'quantity_precision',
          message: 'Línea 2: la cantidad en la unidad base necesita más de 4 decimales.',
          details: { line: '1' },
        }),
      },
    });
    renderApp('/compras/nueva');

    await fillTwoLines();
    await registerPurchase();

    const second = within(lineGroup(2));
    expect(await second.findByText(/la cantidad en la unidad base necesita más de 4 decimales/)).toBeInTheDocument();
    expect(second.getByLabelText('Cantidad')).toHaveAttribute('aria-invalid', 'true');
    expect(within(lineGroup(1)).getByLabelText('Cantidad')).not.toHaveAttribute('aria-invalid', 'true');
  });

  it('un error de línea sin campo propio queda en el producto de la fila', async () => {
    purchasesApi({
      routes: {
        'POST /api/v1/purchases': apiError(422, {
          code: 'variant_inactive',
          message: 'Línea 1: el producto está inactivo.',
          details: { line: '0' },
        }),
      },
    });
    renderApp('/compras/nueva');

    await fillOneLine();
    await registerPurchase();

    expect(await within(lineGroup(1)).findByText(/el producto está inactivo/)).toBeInTheDocument();
  });

  it('no envía la compra sin producto en una línea', async () => {
    const api = purchasesApi();
    renderApp('/compras/nueva');

    await fillInvoice();
    await registerPurchase();

    expect(await within(lineGroup(1)).findByText('Elija el producto')).toBeInTheDocument();
    expect(api.calls('POST /api/v1/purchases')).toHaveLength(0);
  });
});
