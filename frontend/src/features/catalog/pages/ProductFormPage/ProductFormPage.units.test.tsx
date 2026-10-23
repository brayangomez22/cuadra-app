import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json, type Me } from '@/test/fakeApi';
import { aGarment, aProduct, boxOf12, ids, packOf3, plain, unitOf, variantId, withUnits } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { productFormApi } from '../../testing/productForm';

describe('ProductFormPage: unidades de venta', () => {
  const withUnitConversions = (user = aUser()): Me => ({ ...user, modules: ['unit_conversions'] });
  const box = unitOf('caja');
  const unitPath = (unitId: string, variant = ids.pipe) =>
    `/api/v1/products/${ids.pipe}/variants/${variant}/units/${unitId}`;
  const section = async () => within(await screen.findByRole('region', { name: 'Unidades de venta' }));

  it('sin el módulo, la sección no aparece', async () => {
    productFormApi({ [`GET /api/v1/products/${ids.pipe}`]: json(200, withUnits(aProduct(), [boxOf12])) });
    renderApp(`/productos/${ids.pipe}/editar`);

    await screen.findByDisplayValue('10000,00');
    expect(screen.queryByRole('region', { name: 'Unidades de venta' })).not.toBeInTheDocument();
  });

  it('un producto nuevo pide guardarlo antes de agregar unidades', async () => {
    productFormApi({}, withUnitConversions());
    renderApp('/productos/nuevo');

    const units = await section();
    expect(units.getByText(/Guarde el producto para agregar/)).toBeInTheDocument();
    expect(units.queryByRole('button', { name: /^Agregar unidad/ })).not.toBeInTheDocument();
  });

  it('muestra cada unidad con su equivalencia, su precio y su código', async () => {
    productFormApi(
      { [`GET /api/v1/products/${ids.pipe}`]: json(200, withUnits(aProduct(), [boxOf12])) },
      withUnitConversions(),
    );
    renderApp(`/productos/${ids.pipe}/editar`);

    const row = within((await section()).getByRole('listitem', { name: 'caja' }));
    expect(row.getByText('1 caja = 12 und')).toBeInTheDocument();
    expect(plain(row.getByText(/110\.000/).textContent)).toMatch(/\$ 110\.000 sin IVA/);
    expect(row.getByText(boxOf12.barcode!)).toBeInTheDocument();
  });

  it('agrega una unidad con PUT, con la sugerencia proporcional al precio de la base', async () => {
    const api = productFormApi(
      {
        [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
        [`PUT ${unitPath(box.id)}`]: json(200, withUnits(aProduct(), [boxOf12])),
      },
      withUnitConversions(),
    );
    renderApp(`/productos/${ids.pipe}/editar`);

    await userEvent.click((await section()).getByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Agregar unidad de venta' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Unidad de venta' }));
    const options = await screen.findAllByRole('option');
    // The base unit is not an alternate unit.
    expect(options.map((o) => o.textContent)).not.toContain('Unidad (und)');
    await userEvent.click(screen.getByRole('option', { name: 'Caja (caja)' }));
    await userEvent.type(dialog.getByLabelText(/^Cuántas und trae/), '12');
    expect(plain(dialog.getByText(/Sugerido/).textContent)).toContain('$ 120.000');
    await userEvent.type(dialog.getByLabelText('Precio (sin IVA)'), '110000');
    await userEvent.type(dialog.getByLabelText('Código de barras (opcional)'), boxOf12.barcode!);
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar unidad' }));

    await waitFor(() => expect(api.calls(`PUT ${unitPath(box.id)}`)).toHaveLength(1));
    expect(await api.calls(`PUT ${unitPath(box.id)}`)[0]!.json()).toEqual({
      factor: '12',
      price: '110000',
      barcode: boxOf12.barcode,
    });
    expect(await (await section()).findByText('1 caja = 12 und')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('usa la sugerencia proporcional como precio', async () => {
    const api = productFormApi(
      {
        [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
        [`PUT ${unitPath(box.id)}`]: json(200, withUnits(aProduct(), [boxOf12])),
      },
      withUnitConversions(),
    );
    renderApp(`/productos/${ids.pipe}/editar`);

    await userEvent.click((await section()).getByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Agregar unidad de venta' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Unidad de venta' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Caja (caja)' }));
    await userEvent.type(dialog.getByLabelText(/^Cuántas und trae/), '12');
    await userEvent.click(dialog.getByRole('button', { name: 'Usar el sugerido' }));
    expect(dialog.getByLabelText('Precio (sin IVA)')).toHaveValue('120000,00');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar unidad' }));

    await waitFor(() => expect(api.calls(`PUT ${unitPath(box.id)}`)).toHaveLength(1));
    expect(await api.calls(`PUT ${unitPath(box.id)}`)[0]!.json()).toEqual({ factor: '12', price: '120000.00' });
  });

  it('muestra en su campo el código de barras repetido', async () => {
    productFormApi(
      {
        [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
        [`PUT ${unitPath(box.id)}`]: apiError(409, {
          code: 'barcode_taken',
          message: 'El código de barras ya existe.',
        }),
      },
      withUnitConversions(),
    );
    renderApp(`/productos/${ids.pipe}/editar`);

    await userEvent.click((await section()).getByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Agregar unidad de venta' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Unidad de venta' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Caja (caja)' }));
    await userEvent.type(dialog.getByLabelText(/^Cuántas und trae/), '12');
    await userEvent.type(dialog.getByLabelText('Precio (sin IVA)'), '110000');
    await userEvent.type(dialog.getByLabelText('Código de barras (opcional)'), '7700000000001');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar unidad' }));

    const barcode = dialog.getByLabelText('Código de barras (opcional)');
    await waitFor(() => expect(barcode).toHaveAttribute('aria-invalid', 'true'));
    expect(barcode).toHaveAccessibleDescription(expect.stringContaining('El código de barras ya existe.'));
  });

  it('edita el precio de una unidad y la quita con DELETE', async () => {
    const api = productFormApi(
      {
        [`GET /api/v1/products/${ids.pipe}`]: json(200, withUnits(aProduct(), [boxOf12])),
        [`PUT ${unitPath(box.id)}`]: json(200, withUnits(aProduct(), [{ ...boxOf12, price: '105000.00' }])),
        [`DELETE ${unitPath(box.id)}`]: json(200, aProduct()),
      },
      withUnitConversions(),
    );
    renderApp(`/productos/${ids.pipe}/editar`);

    await userEvent.click((await section()).getByRole('button', { name: 'Editar caja' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Editar unidad de venta' }));
    const price = dialog.getByLabelText('Precio (sin IVA)');
    expect(price).toHaveValue('110000,00');
    await userEvent.clear(price);
    await userEvent.type(price, '105000');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar unidad' }));
    await waitFor(() => expect(api.calls(`PUT ${unitPath(box.id)}`)).toHaveLength(1));
    expect(await api.calls(`PUT ${unitPath(box.id)}`)[0]!.json()).toEqual({
      factor: '12',
      price: '105000',
      barcode: boxOf12.barcode,
    });

    await userEvent.click(await (await section()).findByRole('button', { name: 'Quitar caja' }));
    await waitFor(() => expect(api.calls(`DELETE ${unitPath(box.id)}`)).toHaveLength(1));
    expect(await (await section()).findByText(/Sin unidades de venta/)).toBeInTheDocument();
  });

  it('con variantes, agrega la unidad a la variante elegida (paquete de una talla y color)', async () => {
    const red = variantId('M', 'Rojo');
    const garment = aGarment();
    const pack = unitOf('paq');
    const path = `/api/v1/products/${ids.garment}/variants/${red}/units/${pack.id}`;
    const api = productFormApi(
      {
        [`GET /api/v1/products/${ids.garment}`]: json(200, garment),
        [`PUT ${path}`]: json(200, withUnits(garment, [packOf3], red)),
      },
      { ...aUser(), modules: ['variants', 'unit_conversions'] },
    );
    renderApp(`/productos/${ids.garment}/editar`);

    const units = await section();
    await userEvent.click(units.getByRole('button', { name: 'Agregar unidad a Rojo · M' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Agregar unidad de venta' }));
    expect(dialog.getByText('Camiseta básica (Rojo · M)')).toBeInTheDocument();
    await userEvent.click(dialog.getByRole('combobox', { name: 'Unidad de venta' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Paquete (paq)' }));
    await userEvent.type(dialog.getByLabelText(/^Cuántas und trae/), '3');
    await userEvent.type(dialog.getByLabelText('Precio (sin IVA)'), '70000');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar unidad' }));

    await waitFor(() => expect(api.calls(`PUT ${path}`)).toHaveLength(1));
    expect(await api.calls(`PUT ${path}`)[0]!.json()).toEqual({ factor: '3', price: '70000' });
    const group = within(units.getByRole('group', { name: 'Rojo · M' }));
    expect(await group.findByText('1 paq = 3 und')).toBeInTheDocument();
  });
});
