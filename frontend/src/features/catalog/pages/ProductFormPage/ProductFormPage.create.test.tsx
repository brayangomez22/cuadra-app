import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { aProduct, ids, unitOf } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { fillProduct, productFormApi, saveProduct } from '../../testing/productForm';

describe('ProductFormPage: crear un producto', () => {
  it('crea el producto enviando los montos como decimal con punto y vuelve a la lista', async () => {
    const api = productFormApi({ 'POST /api/v1/products': json(201, aProduct()) });
    const { router } = renderApp('/productos/nuevo');

    await fillProduct();
    await saveProduct();

    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toEqual({
      sku: 'tub-pvc-12',
      name: 'Tubo PVC presión 1/2 pulgada',
      base_unit_id: unitOf('und').id,
      cost: '8333.33',
      price: '10000',
      tax_rate: '0.19',
      category_id: null,
      // Without attribute definitions, the product has no values.
      attributes: {},
    });
    expect(await screen.findByText('Producto guardado')).toBeInTheDocument();
  });

  it('envía la categoría, la unidad y la tasa de IVA elegidas', async () => {
    const api = productFormApi({ 'POST /api/v1/products': json(201, aProduct()) });
    renderApp('/productos/nuevo');
    await fillProduct();

    await userEvent.click(screen.getByRole('combobox', { name: 'Categoría' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Plomería' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'Unidad' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Bulto (bulto)' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'IVA' }));
    await userEvent.click(await screen.findByRole('option', { name: /^5\s%$/ }));
    await saveProduct();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({
      category_id: ids.plumbing,
      base_unit_id: unitOf('bulto').id,
      tax_rate: '0.05',
    });
  });

  it('ofrece las unidades de la empresa', async () => {
    productFormApi();
    renderApp('/productos/nuevo');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Unidad' }));

    const options = await screen.findAllByRole('option');
    expect(options.map((o) => o.textContent)).toEqual([
      'Bulto (bulto)',
      'Caja (caja)',
      'Metro (m)',
      'Paquete (paq)',
      'Rollo (rollo)',
      'Unidad (und)',
    ]);
  });

  it('un producto nuevo propone el IVA por defecto de la empresa', async () => {
    const api = productFormApi(
      { 'POST /api/v1/products': json(201, aProduct()) },
      { ...aUser(), business_template: 'drogueria', default_tax_rate: '0' },
    );
    renderApp('/productos/nuevo');

    await fillProduct();
    expect(screen.getByRole('combobox', { name: 'IVA' })).toHaveTextContent(/^0\s%$/);
    await saveProduct();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({ tax_rate: '0' });
  });

  it('con variants inactivo, el formulario es el de un producto simple', async () => {
    const api = productFormApi();
    renderApp('/productos/nuevo');

    await screen.findByLabelText('SKU');

    expect(screen.queryByRole('heading', { name: 'Variantes' })).not.toBeInTheDocument();
    expect(screen.getByLabelText('Código de barras (opcional)')).toBeInTheDocument();
    expect(api.calls('GET /api/v1/variant-axes')).toHaveLength(0);
  });

  it('quien no puede editar el catálogo ve que no tiene permiso', async () => {
    const api = productFormApi({ ...signedIn(aUser('cashier')) });
    renderApp('/productos/nuevo');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
    expect(screen.queryByLabelText('SKU')).not.toBeInTheDocument();
    expect(api.calls('GET /api/v1/categories')).toHaveLength(0);
  });
});
