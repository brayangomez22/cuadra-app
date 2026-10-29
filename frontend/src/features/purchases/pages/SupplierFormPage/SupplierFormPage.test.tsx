import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aTextileSupplier, supplierIds } from '../../testing/fixtures';
import { purchasesApi } from '../../testing/purchasesApi';

async function fillSupplier(nit: string) {
  await userEvent.type(await screen.findByLabelText('Nombre o razón social'), 'Ferretería Mayorista del Sur');
  await userEvent.type(screen.getByLabelText('NIT'), nit);
  await userEvent.click(screen.getByRole('button', { name: 'Guardar proveedor' }));
}

describe('SupplierFormPage', () => {
  it('crea un proveedor con su plazo y vuelve a la lista', async () => {
    const api = purchasesApi({
      routes: { 'POST /api/v1/suppliers': json(201, aTextileSupplier({ name: 'Ferretería Mayorista del Sur' })) },
    });
    renderApp('/compras/proveedores/nuevo');

    await userEvent.type(await screen.findByLabelText('Nombre o razón social'), 'Ferretería Mayorista del Sur');
    await userEvent.type(screen.getByLabelText('NIT'), '901.555.444-2');
    await userEvent.clear(screen.getByLabelText('Plazo de pago'));
    await userEvent.type(screen.getByLabelText('Plazo de pago'), '15');
    await userEvent.type(screen.getByLabelText('Correo (opcional)'), 'ventas@mayoristasur.co');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar proveedor' }));

    expect(await screen.findByRole('heading', { name: 'Proveedores' })).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/suppliers')[0]?.json()).toEqual({
      name: 'Ferretería Mayorista del Sur',
      nit: '901.555.444-2',
      email: 'ventas@mayoristasur.co',
      payment_term_days: 15,
      active: true,
    });
  });

  it('muestra el error del DV junto al NIT', async () => {
    purchasesApi({
      routes: {
        'POST /api/v1/suppliers': apiError(422, {
          code: 'invalid_nit_check_digit',
          message: 'El dígito de verificación del NIT no coincide.',
        }),
      },
    });
    renderApp('/compras/proveedores/nuevo');

    await fillSupplier('901555444-9');

    const nit = screen.getByLabelText('NIT');
    expect(await screen.findByText('El dígito de verificación del NIT no coincide.')).toBeInTheDocument();
    expect(nit).toHaveAttribute('aria-invalid', 'true');
    expect(nit).toHaveFocus();
  });

  it('el NIT repetido se marca en el NIT', async () => {
    purchasesApi({
      routes: {
        'POST /api/v1/suppliers': apiError(409, {
          code: 'supplier_nit_taken',
          message: 'Ya existe un proveedor con el NIT 800765432-1.',
        }),
      },
    });
    renderApp('/compras/proveedores/nuevo');

    await fillSupplier('800765432-1');

    expect(await screen.findByText('Ya existe un proveedor con el NIT 800765432-1.')).toBeInTheDocument();
    expect(screen.getByLabelText('NIT')).toHaveAttribute('aria-invalid', 'true');
  });

  it('al editar muestra lo que se le debe y envía los cambios', async () => {
    const api = purchasesApi({
      routes: {
        [`GET /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor()),
        [`PUT /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor({ active: false })),
      },
    });
    renderApp(`/compras/proveedores/${supplierIds.hardware}/editar`);

    expect(plain((await screen.findByText(/Se le debe/)).textContent)).toBe(
      'Distribuidora Andina SAS · Se le debe $ 318.480',
    );
    expect(screen.getByLabelText('NIT')).toHaveValue('800765432-1');
    await userEvent.click(screen.getByRole('checkbox', { name: 'Activo' }));
    await userEvent.click(screen.getByRole('button', { name: 'Guardar proveedor' }));

    await screen.findByRole('heading', { name: 'Proveedores' });
    expect(await api.calls(`PUT /api/v1/suppliers/${supplierIds.hardware}`)[0]?.json()).toMatchObject({
      nit: '800765432-1',
      contact_name: 'Jorge Peña',
      payment_term_days: 30,
      active: false,
    });
  });
});
