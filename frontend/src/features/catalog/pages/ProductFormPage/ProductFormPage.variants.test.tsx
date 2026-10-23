import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { aGarment, ids, variantId } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { garmentServer } from '../../testing/garmentServer';
import { fillGarment, productFormApi, saveProduct, withVariants } from '../../testing/productForm';

describe('ProductFormPage: variantes', () => {
  it('solo ofrece los ejes y valores definidos por la empresa, sin texto libre', async () => {
    productFormApi({}, withVariants());
    renderApp('/productos/nuevo');

    const size = within(await screen.findByRole('group', { name: 'Talla' }));
    const color = within(screen.getByRole('group', { name: 'Color' }));

    expect(size.getAllByRole('checkbox')).toHaveLength(3);
    for (const value of ['S', 'M', 'L']) expect(size.getByRole('checkbox', { name: value })).not.toBeChecked();
    expect(color.getAllByRole('checkbox')).toHaveLength(2);
    for (const value of ['Rojo', 'Azul']) expect(color.getByRole('checkbox', { name: value })).toBeInTheDocument();
    expect(within(screen.getByRole('region', { name: 'Variantes' })).queryByRole('textbox')).not.toBeInTheDocument();
  });

  it('con variants activo, 3 tallas × 2 colores crean 6 variantes (1 PUT a la por defecto + 5 POST)', async () => {
    const api = productFormApi(garmentServer(), withVariants());
    const { router } = renderApp('/productos/nuevo');

    await fillGarment();
    const matrix = within(screen.getByRole('table', { name: 'Variantes del producto' }));
    expect(matrix.getAllByRole('row')).toHaveLength(7); // header + 6
    expect(matrix.getByLabelText('SKU de Azul · M')).toHaveAttribute('placeholder', 'CAM-M-AZUL');
    await userEvent.type(matrix.getByLabelText('Precio de Rojo · L'), '27000');
    await userEvent.type(matrix.getByLabelText('Código de barras de Azul · M'), '7709990000004');
    await saveProduct();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({
      sku: 'CAM-S-ROJO',
      name: 'Camiseta básica',
      cost: '15000',
      price: '25000',
    });
    expect(
      await api.calls(`PUT /api/v1/products/${ids.garment}/variants/${variantId('S', 'Rojo')}`)[0]?.json(),
    ).toEqual({
      sku: 'CAM-S-ROJO',
      cost: '15000',
      price: '25000',
      axis_values: { talla: 'S', color: 'Rojo' },
      active: true,
    });
    const added = await Promise.all(api.calls(`POST /api/v1/products/${ids.garment}/variants`).map((r) => r.json()));
    expect(added).toEqual([
      { sku: 'CAM-S-AZUL', cost: '15000', price: '25000', axis_values: { talla: 'S', color: 'Azul' } },
      { sku: 'CAM-M-ROJO', cost: '15000', price: '25000', axis_values: { talla: 'M', color: 'Rojo' } },
      {
        sku: 'CAM-M-AZUL',
        barcode: '7709990000004',
        cost: '15000',
        price: '25000',
        axis_values: { talla: 'M', color: 'Azul' },
      },
      { sku: 'CAM-L-ROJO', cost: '15000', price: '27000', axis_values: { talla: 'L', color: 'Rojo' } },
      { sku: 'CAM-L-AZUL', cost: '15000', price: '25000', axis_values: { talla: 'L', color: 'Azul' } },
    ]);
  });

  it('si falla una variante, queda editando el producto y reintentar solo envía las que faltan', async () => {
    const api = productFormApi(garmentServer({ failSku: 'CAM-M-AZUL' }), withVariants());
    const { router } = renderApp('/productos/nuevo');
    await fillGarment();

    await saveProduct();

    expect(await screen.findByText(/El producto se guardó, pero 1 variante no/)).toBeInTheDocument();
    const matrix = within(screen.getByRole('table', { name: 'Variantes del producto' }));
    expect(matrix.getByLabelText('SKU de Azul · M')).toHaveAccessibleDescription(
      expect.stringContaining('Ya existe un producto con el SKU CAM-M-AZUL'),
    );
    expect(router.state.location.pathname).toBe('/productos/nuevo');
    expect(api.calls(`POST /api/v1/products/${ids.garment}/variants`)).toHaveLength(5);

    await userEvent.type(matrix.getByLabelText('SKU de Azul · M'), 'CAM-M-AZUL-2');
    await saveProduct();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(api.calls('POST /api/v1/products')).toHaveLength(1);
    expect(api.calls(`PUT /api/v1/products/${ids.garment}`)).toHaveLength(1);
    expect(api.calls(`PUT /api/v1/products/${ids.garment}/variants/${variantId('S', 'Rojo')}`)).toHaveLength(1);
    const posts = api.calls(`POST /api/v1/products/${ids.garment}/variants`);
    expect(posts).toHaveLength(6);
    expect(await posts.at(-1)!.json()).toMatchObject({
      sku: 'CAM-M-AZUL-2',
      axis_values: { talla: 'M', color: 'Azul' },
    });
  });

  it('al editar, solo envía las variantes que cambian', async () => {
    const api = productFormApi(
      {
        [`GET /api/v1/products/${ids.garment}`]: json(200, aGarment()),
        [`PUT /api/v1/products/${ids.garment}`]: json(200, aGarment()),
        [`PUT /api/v1/products/${ids.garment}/variants/${variantId('L', 'Rojo')}`]: json(200, aGarment()),
      },
      withVariants(),
    );
    const { router } = renderApp(`/productos/${ids.garment}/editar`);

    const matrix = within(await screen.findByRole('table', { name: 'Variantes del producto' }));
    expect(matrix.getByLabelText('SKU de Rojo · L')).toHaveValue('CAM-L-ROJO');
    const price = matrix.getByLabelText('Precio de Rojo · L');
    await userEvent.clear(price);
    await userEvent.type(price, '28000');
    await userEvent.click(matrix.getByRole('checkbox', { name: 'Activa: Rojo · L' }));
    // The values of the existing variants cannot be taken off: variants are not deleted.
    expect(within(screen.getByRole('group', { name: 'Talla' })).getByRole('checkbox', { name: 'M' })).toBeDisabled();
    await saveProduct();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(
      await api.calls(`PUT /api/v1/products/${ids.garment}/variants/${variantId('L', 'Rojo')}`)[0]?.json(),
    ).toEqual({
      sku: 'CAM-L-ROJO',
      barcode: '770999000605',
      cost: '15000.00',
      price: '28000',
      axis_values: { talla: 'L', color: 'Rojo' },
      active: false,
    });
    expect(api.calls(`POST /api/v1/products/${ids.garment}/variants`)).toHaveLength(0);
  });

  it('con el módulo apagado, un producto con variantes las muestra sin editarlas', async () => {
    productFormApi({ [`GET /api/v1/products/${ids.garment}`]: json(200, aGarment()) });
    renderApp(`/productos/${ids.garment}/editar`);

    const variants = within(await screen.findByRole('region', { name: 'Variantes' }));

    expect(variants.getByText(/Active el módulo de variantes para editarlas/)).toBeInTheDocument();
    expect(variants.getByRole('row', { name: /Azul · M/ })).toHaveTextContent('CAM-M-AZUL');
    expect(variants.queryByRole('textbox')).not.toBeInTheDocument();
    expect(variants.queryByRole('checkbox')).not.toBeInTheDocument();
  });
});
