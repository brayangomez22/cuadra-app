import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aProduct, ids, page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const categories = { items: [{ id: ids.plumbing, name: 'Plomería', created_at: '2026-10-01T15:00:00Z' }] };

function catalogApi(extra: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(),
    'GET /api/v1/categories': json(200, categories),
    'GET /api/v1/products': json(200, page([])),
    ...extra,
  });
}

async function fillProduct({ cost = '8333,33', price = '10000' }: { cost?: string; price?: string } = {}) {
  await userEvent.type(await screen.findByLabelText('SKU'), 'tub-pvc-12');
  await userEvent.type(screen.getByLabelText('Nombre'), 'Tubo PVC presión 1/2 pulgada');
  if (cost) await userEvent.type(screen.getByLabelText('Costo'), cost);
  if (price) await userEvent.type(screen.getByLabelText('Precio de venta (sin IVA)'), price);
}

const save = () => userEvent.click(screen.getByRole('button', { name: 'Guardar producto' }));

describe('ProductFormPage', () => {
  it('no envía el formulario con el precio vacío', async () => {
    const api = catalogApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '' });
    await save();

    const price = screen.getByLabelText('Precio de venta (sin IVA)');
    expect(price).toHaveAttribute('aria-invalid', 'true');
    expect(price).toHaveAccessibleDescription(expect.stringContaining('Escriba el precio de venta'));
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('no envía un precio negativo', async () => {
    const api = catalogApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '-500' });
    await save();

    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toHaveAccessibleDescription(
      expect.stringContaining('El precio debe ser mayor que cero'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('rechaza un precio de cero', async () => {
    const api = catalogApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '0,00' });
    await save();

    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toHaveAccessibleDescription(
      expect.stringContaining('El precio debe ser mayor que cero'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('no envía un costo negativo', async () => {
    const api = catalogApi();
    renderApp('/productos/nuevo');

    await fillProduct({ cost: '-1' });
    await save();

    expect(screen.getByLabelText('Costo')).toHaveAccessibleDescription(
      expect.stringContaining('El costo no puede ser negativo'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('rechaza un monto con punto de miles ambiguo', async () => {
    const api = catalogApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '12.500' });
    await save();

    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toHaveAccessibleDescription(
      expect.stringContaining('Escriba el valor sin puntos de miles: 12500'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('crea el producto enviando los montos como decimal con punto y vuelve a la lista', async () => {
    const api = catalogApi({ 'POST /api/v1/products': json(201, aProduct()) });
    const { router } = renderApp('/productos/nuevo');

    await fillProduct();
    await save();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toEqual({
      sku: 'tub-pvc-12',
      name: 'Tubo PVC presión 1/2 pulgada',
      base_unit: 'und',
      cost: '8333.33',
      price: '10000',
      tax_rate: '0.19',
      category_id: null,
    });
    expect(await screen.findByText('Producto guardado')).toBeInTheDocument();
  });

  it('envía la categoría, la unidad y la tasa de IVA elegidas', async () => {
    const api = catalogApi({ 'POST /api/v1/products': json(201, aProduct()) });
    renderApp('/productos/nuevo');
    await fillProduct();

    await userEvent.click(screen.getByRole('combobox', { name: 'Categoría' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Plomería' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'Unidad' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Metro (m)' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'IVA' }));
    await userEvent.click(await screen.findByRole('option', { name: /^5\s%$/ }));
    await save();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({
      category_id: ids.plumbing,
      base_unit: 'm',
      tax_rate: '0.05',
    });
  });

  it('muestra el SKU repetido en su campo', async () => {
    catalogApi({
      'POST /api/v1/products': apiError(409, 'sku_taken', 'Ya existe un producto con el SKU TUB-PVC-12'),
    });
    renderApp('/productos/nuevo');

    await fillProduct();
    await save();

    await waitFor(() =>
      expect(screen.getByLabelText('SKU')).toHaveAccessibleDescription(
        expect.stringContaining('Ya existe un producto con el SKU TUB-PVC-12'),
      ),
    );
  });

  it('precarga el producto y guarda los cambios con PUT', async () => {
    const api = catalogApi({
      [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct({ barcode: '7701234567890' })),
      [`PUT /api/v1/products/${ids.pipe}`]: json(200, aProduct({ price: '10500.00' })),
    });
    const { router } = renderApp(`/productos/${ids.pipe}/editar`);

    // Amounts come back with a decimal comma: "1.125" would look like a thousands dot.
    const price = await screen.findByDisplayValue('10000,00');
    expect(screen.getByLabelText('SKU')).toHaveValue('TUB-PVC-12');
    expect(screen.getByLabelText('Código de barras')).toHaveValue('7701234567890');
    await userEvent.clear(price);
    await userEvent.type(price, '10500');
    await save();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(await api.calls(`PUT /api/v1/products/${ids.pipe}`)[0]?.json()).toEqual({
      sku: 'TUB-PVC-12',
      barcode: '7701234567890',
      name: 'Tubo PVC presión 1/2 pulgada',
      base_unit: 'und',
      cost: '8333.33',
      price: '10500',
      tax_rate: '0.19',
      category_id: null,
    });
  });

  it('desactiva un producto y bloquea el cambio de precio hasta reactivarlo', async () => {
    const api = catalogApi({
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

  it('quien no puede editar el catálogo ve que no tiene permiso', async () => {
    const api = catalogApi({ ...signedIn(aUser('cashier')) });
    renderApp('/productos/nuevo');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
    expect(screen.queryByLabelText('SKU')).not.toBeInTheDocument();
    expect(api.calls('GET /api/v1/categories')).toHaveLength(0);
  });
});
