import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import {
  aGarment,
  aProduct,
  attributes,
  axes,
  ids,
  page,
  unitOf,
  units,
  variantId,
  type Product,
  type ProductVariant,
} from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const categories = { items: [{ id: ids.plumbing, name: 'Plomería', created_at: '2026-10-01T15:00:00Z' }] };

function catalogApi(extra: Parameters<typeof fakeApi>[0] = {}, user: Me = aUser()) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/categories': json(200, categories),
    'GET /api/v1/products': json(200, page([])),
    'GET /api/v1/units': json(200, { items: units }),
    'GET /api/v1/product-attributes': json(200, { items: [] }),
    'GET /api/v1/variant-axes': json(200, { items: axes }),
    ...extra,
  });
}

const withVariants = (user = aUser()): Me => ({ ...user, modules: ['variants'] });

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
    const api = catalogApi({ 'POST /api/v1/products': json(201, aProduct()) });
    renderApp('/productos/nuevo');
    await fillProduct();

    await userEvent.click(screen.getByRole('combobox', { name: 'Categoría' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Plomería' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'Unidad' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Bulto (bulto)' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'IVA' }));
    await userEvent.click(await screen.findByRole('option', { name: /^5\s%$/ }));
    await save();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({
      category_id: ids.plumbing,
      base_unit_id: unitOf('bulto').id,
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
    expect(screen.getByLabelText('Código de barras (opcional)')).toHaveValue('7701234567890');
    await userEvent.clear(price);
    await userEvent.type(price, '10500');
    await save();

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

  it('ofrece las unidades de la empresa', async () => {
    catalogApi();
    renderApp('/productos/nuevo');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Unidad' }));

    const options = await screen.findAllByRole('option');
    expect(options.map((o) => o.textContent)).toEqual(['Bulto (bulto)', 'Metro (m)', 'Unidad (und)']);
  });

  it('un producto nuevo propone el IVA por defecto de la empresa', async () => {
    const api = catalogApi(
      { 'POST /api/v1/products': json(201, aProduct()) },
      { ...aUser(), business_template: 'drogueria', default_tax_rate: '0' },
    );
    renderApp('/productos/nuevo');

    await fillProduct();
    expect(screen.getByRole('combobox', { name: 'IVA' })).toHaveTextContent(/^0\s%$/);
    await save();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({ tax_rate: '0' });
  });

  it('con variants inactivo, el formulario es el de un producto simple', async () => {
    const api = catalogApi();
    renderApp('/productos/nuevo');

    await screen.findByLabelText('SKU');

    expect(screen.queryByRole('heading', { name: 'Variantes' })).not.toBeInTheDocument();
    expect(screen.getByLabelText('Código de barras (opcional)')).toBeInTheDocument();
    expect(api.calls('GET /api/v1/variant-axes')).toHaveLength(0);
  });

  describe('atributos', () => {
    const attributesApi = (extra: Parameters<typeof fakeApi>[0] = {}) =>
      catalogApi({
        'GET /api/v1/product-attributes': json(200, { items: attributes }),
        'POST /api/v1/products': json(201, aProduct()),
        ...extra,
      });

    it('envía los atributos de la empresa con el valor de cada tipo', async () => {
      const api = attributesApi();
      renderApp('/productos/nuevo');
      await fillProduct();

      await userEvent.type(screen.getByLabelText('Marca'), 'Acme');
      await userEvent.click(screen.getByRole('combobox', { name: /^Material/ }));
      await userEvent.click(await screen.findByRole('option', { name: 'PVC' }));
      await userEvent.type(screen.getByLabelText('Peso (opcional)'), '2,5');
      await userEvent.click(screen.getByRole('checkbox', { name: 'Importado' }));
      await save();

      await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
      expect((await api.calls('POST /api/v1/products')[0]!.json()).attributes).toEqual({
        marca: 'Acme',
        material: 'PVC',
        peso: '2.5',
        importado: true,
      });
    });

    it('omite los atributos opcionales vacíos y envía el sí/no aunque no se marque', async () => {
      const api = attributesApi();
      renderApp('/productos/nuevo');
      await fillProduct();

      await userEvent.type(screen.getByLabelText('Marca'), 'Acme');
      await save();

      await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
      expect((await api.calls('POST /api/v1/products')[0]!.json()).attributes).toEqual({ marca: 'Acme', importado: false });
    });

    it('no envía el producto sin un atributo obligatorio ni con un número inválido', async () => {
      const api = attributesApi();
      renderApp('/productos/nuevo');
      await fillProduct();

      await userEvent.type(screen.getByLabelText('Peso (opcional)'), 'dos');
      await save();

      expect(screen.getByLabelText('Marca')).toHaveAccessibleDescription(expect.stringContaining('Este dato es obligatorio'));
      expect(screen.getByLabelText('Peso (opcional)')).toHaveAccessibleDescription(
        expect.stringContaining('Escriba un número'),
      );
      expect(api.calls('POST /api/v1/products')).toHaveLength(0);
    });

    it('muestra el error de la API en el atributo que nombra details.attribute', async () => {
      attributesApi({
        'POST /api/v1/products': apiError(422, 'invalid_attribute_value', 'El peso debe ser un número', {
          attribute: 'peso',
        }),
      });
      renderApp('/productos/nuevo');
      await fillProduct();
      await userEvent.type(screen.getByLabelText('Marca'), 'Acme');
      await userEvent.type(screen.getByLabelText('Peso (opcional)'), '2');

      await save();

      await waitFor(() =>
        expect(screen.getByLabelText('Peso (opcional)')).toHaveAccessibleDescription(
          expect.stringContaining('El peso debe ser un número'),
        ),
      );
    });

    it('precarga los atributos del producto al editar', async () => {
      attributesApi({
        [`GET /api/v1/products/${ids.pipe}`]: json(
          200,
          aProduct({ attributes: { marca: 'Acme', material: 'PVC', peso: '1.25', importado: true } }),
        ),
      });
      renderApp(`/productos/${ids.pipe}/editar`);

      expect(await screen.findByLabelText('Marca')).toHaveValue('Acme');
      expect(screen.getByRole('combobox', { name: /^Material/ })).toHaveTextContent('PVC');
      expect(screen.getByLabelText('Peso (opcional)')).toHaveValue('1,25');
      expect(screen.getByRole('checkbox', { name: 'Importado' })).toBeChecked();
    });
  });

  describe('variantes', () => {
    // garmentServer answers like the API, keeping the product it creates: a
    // new product has only its default variant; adding variants grows it.
    function garmentServer({ failSku }: { failSku?: string } = {}) {
      let product: Product | undefined;
      const label = (values: Record<string, string>) =>
        Object.keys(values)
          .sort()
          .map((k) => values[k])
          .join(' · ');
      const fromBody = (body: Record<string, unknown>, id: string, isDefault: boolean): ProductVariant => {
        const values = body.axis_values as Record<string, string>;
        return {
          id,
          sku: String(body.sku).toUpperCase(),
          price: String(body.price),
          price_with_tax: String(body.price),
          cost: String(body.cost),
          axis_values: values,
          label: label(values),
          units: [],
          is_default: isDefault,
          active: body.active !== false,
          created_at: '2026-10-17T15:00:00Z',
        };
      };
      const routes: Parameters<typeof fakeApi>[0] = {
        'POST /api/v1/products': async (request) => {
          const body = await request.json();
          product = aGarment({ name: body.name, variants: [] });
          product.variants = [fromBody({ ...body, axis_values: {} }, variantId('S', 'Rojo'), true)];
          return json(201, product);
        },
        [`PUT /api/v1/products/${ids.garment}`]: async (request) => {
          const body = await request.json();
          const [first, ...rest] = product!.variants;
          product = { ...product!, variants: [{ ...first!, sku: body.sku }, ...rest] };
          return json(200, product);
        },
        [`PUT /api/v1/products/${ids.garment}/variants/${variantId('S', 'Rojo')}`]: async (request) => {
          const body = await request.json();
          const [, ...rest] = product!.variants;
          product = { ...product!, variants: [fromBody(body, variantId('S', 'Rojo'), true), ...rest] };
          return json(200, product);
        },
        [`POST /api/v1/products/${ids.garment}/variants`]: async (request) => {
          const body = await request.json();
          if (body.sku === failSku) return apiError(409, 'sku_taken', `Ya existe un producto con el SKU ${failSku}`);
          const values = body.axis_values as Record<string, string>;
          const variant = fromBody(body, variantId(values.talla!, values.color!), false);
          product = { ...product!, variants: [...product!.variants, variant] };
          return json(201, product);
        },
      };
      return routes;
    }

    async function chooseValues(axis: string, values: string[]) {
      const group = within(await screen.findByRole('group', { name: axis }));
      for (const value of values) await userEvent.click(group.getByRole('checkbox', { name: value }));
    }

    async function fillGarment() {
      await userEvent.type(await screen.findByLabelText('SKU'), 'cam');
      await userEvent.type(screen.getByLabelText('Nombre'), 'Camiseta básica');
      await userEvent.type(screen.getByLabelText('Costo'), '15000');
      await userEvent.type(screen.getByLabelText('Precio de venta (sin IVA)'), '25000');
      await chooseValues('Talla', ['S', 'M', 'L']);
      await chooseValues('Color', ['Rojo', 'Azul']);
    }

    it('solo ofrece los ejes y valores definidos por la empresa, sin texto libre', async () => {
      catalogApi({}, withVariants());
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
      const api = catalogApi(garmentServer(), withVariants());
      const { router } = renderApp('/productos/nuevo');

      await fillGarment();
      const matrix = within(screen.getByRole('table', { name: 'Variantes del producto' }));
      expect(matrix.getAllByRole('row')).toHaveLength(7); // header + 6
      expect(matrix.getByLabelText('SKU de Azul · M')).toHaveAttribute('placeholder', 'CAM-M-AZUL');
      await userEvent.type(matrix.getByLabelText('Precio de Rojo · L'), '27000');
      await userEvent.type(matrix.getByLabelText('Código de barras de Azul · M'), '7709990000004');
      await save();

      await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
      expect(await api.calls('POST /api/v1/products')[0]?.json()).toMatchObject({
        sku: 'CAM-S-ROJO',
        name: 'Camiseta básica',
        cost: '15000',
        price: '25000',
      });
      expect(await api.calls(`PUT /api/v1/products/${ids.garment}/variants/${variantId('S', 'Rojo')}`)[0]?.json()).toEqual(
        { sku: 'CAM-S-ROJO', cost: '15000', price: '25000', axis_values: { talla: 'S', color: 'Rojo' }, active: true },
      );
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
      const api = catalogApi(garmentServer({ failSku: 'CAM-M-AZUL' }), withVariants());
      const { router } = renderApp('/productos/nuevo');
      await fillGarment();

      await save();

      expect(await screen.findByText(/El producto se guardó, pero 1 variante no/)).toBeInTheDocument();
      const matrix = within(screen.getByRole('table', { name: 'Variantes del producto' }));
      expect(matrix.getByLabelText('SKU de Azul · M')).toHaveAccessibleDescription(
        expect.stringContaining('Ya existe un producto con el SKU CAM-M-AZUL'),
      );
      expect(router.state.location.pathname).toBe('/productos/nuevo');
      expect(api.calls(`POST /api/v1/products/${ids.garment}/variants`)).toHaveLength(5);

      await userEvent.type(matrix.getByLabelText('SKU de Azul · M'), 'CAM-M-AZUL-2');
      await save();

      await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
      expect(api.calls('POST /api/v1/products')).toHaveLength(1);
      expect(api.calls(`PUT /api/v1/products/${ids.garment}`)).toHaveLength(1);
      expect(api.calls(`PUT /api/v1/products/${ids.garment}/variants/${variantId('S', 'Rojo')}`)).toHaveLength(1);
      const posts = api.calls(`POST /api/v1/products/${ids.garment}/variants`);
      expect(posts).toHaveLength(6);
      expect(await posts.at(-1)!.json()).toMatchObject({ sku: 'CAM-M-AZUL-2', axis_values: { talla: 'M', color: 'Azul' } });
    });

    it('al editar, solo envía las variantes que cambian', async () => {
      const api = catalogApi(
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
      await save();

      await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
      expect(await api.calls(`PUT /api/v1/products/${ids.garment}/variants/${variantId('L', 'Rojo')}`)[0]?.json()).toEqual({
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
      catalogApi({ [`GET /api/v1/products/${ids.garment}`]: json(200, aGarment()) });
      renderApp(`/productos/${ids.garment}/editar`);

      const variants = within(await screen.findByRole('region', { name: 'Variantes' }));

      expect(variants.getByText(/Active el módulo de variantes para editarlas/)).toBeInTheDocument();
      expect(variants.getByRole('row', { name: /Azul · M/ })).toHaveTextContent('CAM-M-AZUL');
      expect(variants.queryByRole('textbox')).not.toBeInTheDocument();
      expect(variants.queryByRole('checkbox')).not.toBeInTheDocument();
    });
  });

  it('quien no puede editar el catálogo ve que no tiene permiso', async () => {
    const api = catalogApi({ ...signedIn(aUser('cashier')) });
    renderApp('/productos/nuevo');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
    expect(screen.queryByLabelText('SKU')).not.toBeInTheDocument();
    expect(api.calls('GET /api/v1/categories')).toHaveLength(0);
  });
});
