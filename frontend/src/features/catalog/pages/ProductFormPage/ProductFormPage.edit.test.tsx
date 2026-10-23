import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { aProduct, ids, unitOf } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { productFormApi, saveProduct } from '../../testing/productForm';

describe('ProductFormPage: editar un producto', () => {
  it('precarga el producto y guarda los cambios con PUT', async () => {
    const api = productFormApi({
      [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct({ barcode: '7701234567890' })),
      [`PUT /api/v1/products/${ids.pipe}`]: json(200, aProduct({ price: '10500.00' })),
    });
    const { router } = renderApp(`/productos/${ids.pipe}/editar`);

    // Amounts come back with a decimal comma: "1.125" would look like a thousands dot.
    const price = await screen.findByDisplayValue('10000,00');
    expect(screen.getByLabelText('SKU')).toHaveValue('TUB-PVC-12');
    expect(screen.getByLabelText('Código de barras (opcional)')).toHaveValue('7701234567890');
    await userEvent.clear(price);
    await userEvent.type(price, '10500');
    await saveProduct();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(await api.calls(`PUT /api/v1/products/${ids.pipe}`)[0]?.json()).toEqual({
      sku: 'TUB-PVC-12',
      barcode: '7701234567890',
      name: 'Tubo PVC presión 1/2 pulgada',
      base_unit_id: unitOf('und').id,
      cost: '8333.33',
      price: '10500',
      tax_rate: '0.19',
      category_id: null,
      attributes: {},
    });
  });

  it('desactiva un producto y bloquea el cambio de precio hasta reactivarlo', async () => {
    const api = productFormApi({
      [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
      [`POST /api/v1/products/${ids.pipe}/deactivate`]: json(200, aProduct({ active: false })),
    });
    renderApp(`/productos/${ids.pipe}/editar`);

    await userEvent.click(await screen.findByRole('button', { name: 'Desactivar' }));

    expect(await screen.findByRole('button', { name: 'Activar' })).toBeInTheDocument();
    expect(api.calls(`POST /api/v1/products/${ids.pipe}/deactivate`)).toHaveLength(1);
    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toBeDisabled();
    expect(screen.getByText('Inactivo')).toBeInTheDocument();
  });
});
