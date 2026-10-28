import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { attributes, page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { attributesApi } from '../../testing/settingsApi';

async function chooseType(dialog: { getByRole: typeof screen.getByRole }, type: string) {
  await userEvent.click(dialog.getByRole('combobox', { name: 'Tipo' }));
  await userEvent.click(await screen.findByRole('option', { name: type }));
}

describe('AttributesSettings', () => {
  it('lista los atributos con su tipo y si son obligatorios', async () => {
    attributesApi();
    renderApp('/configuracion/atributos');

    const row = within((await screen.findByRole('cell', { name: 'Material' })).closest('tr')!);
    expect(row.getByRole('cell', { name: 'Lista: Algodón, PVC' })).toBeInTheDocument();
    const brand = within(screen.getByRole('cell', { name: 'Marca' }).closest('tr')!);
    expect(brand.getByText('Obligatorio')).toBeInTheDocument();
  });

  it('crea un atributo de lista con sus opciones, una por línea', async () => {
    // Droguería: la forma farmacéutica.
    const created = {
      id: '01920000-0000-7000-8000-000000000809',
      key: 'forma',
      label: 'Forma farmacéutica',
      type: 'list',
      options: ['Tableta', 'Jarabe'],
      required: false,
    };
    const api = attributesApi({ 'POST /api/v1/product-attributes': json(201, created) });
    renderApp('/configuracion/atributos');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar atributo' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Agregar atributo' }));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Forma farmacéutica');
    await userEvent.type(dialog.getByLabelText('Clave'), 'forma');
    await chooseType(dialog, 'Lista');
    await userEvent.type(dialog.getByLabelText('Opciones'), 'Tableta{Enter}Jarabe{Enter}');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls('POST /api/v1/product-attributes')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/product-attributes')[0]!.json()).toEqual({
      key: 'forma',
      label: 'Forma farmacéutica',
      type: 'list',
      options: ['Tableta', 'Jarabe'],
      required: false,
    });
  });

  it('un atributo de texto se crea sin opciones y obligatorio si se marca', async () => {
    // Ferretería: la marca.
    const api = attributesApi({ 'POST /api/v1/product-attributes': json(201, attributes[0]) });
    renderApp('/configuracion/atributos');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar atributo' }));
    const dialog = within(screen.getByRole('dialog'));
    expect(dialog.queryByLabelText('Opciones')).not.toBeInTheDocument();
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Marca');
    await userEvent.type(dialog.getByLabelText('Clave'), 'marca');
    await userEvent.click(dialog.getByLabelText('Obligatorio'));
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls('POST /api/v1/product-attributes')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/product-attributes')[0]!.json()).toEqual({
      key: 'marca',
      label: 'Marca',
      type: 'text',
      required: true,
    });
  });

  it('valida la clave antes de enviar', async () => {
    const api = attributesApi();
    renderApp('/configuracion/atributos');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar atributo' }));
    const dialog = within(screen.getByRole('dialog'));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Principio activo');
    await userEvent.type(dialog.getByLabelText('Clave'), 'Principio activo');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(dialog.getByLabelText('Clave')).toHaveAttribute('aria-invalid', 'true');
    expect(api.calls('POST /api/v1/product-attributes')).toHaveLength(0);
  });

  it('al editar, la clave y el tipo no cambian y se envían la etiqueta, las opciones y si es obligatorio', async () => {
    const material = attributes[1]!;
    const api = attributesApi({
      [`PUT /api/v1/product-attributes/${material.id}`]: json(200, {
        ...material,
        options: ['Algodón', 'PVC', 'Cobre'],
      }),
    });
    renderApp('/configuracion/atributos');

    const row = within((await screen.findByRole('cell', { name: 'Material' })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Editar Material' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Editar atributo' }));
    expect(dialog.getByLabelText('Clave')).toHaveAttribute('readonly');
    expect(dialog.getByRole('combobox', { name: 'Tipo' })).toBeDisabled();
    await userEvent.type(dialog.getByLabelText('Opciones'), '{Enter}Cobre');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls(`PUT /api/v1/product-attributes/${material.id}`)).toHaveLength(1));
    expect(await api.calls(`PUT /api/v1/product-attributes/${material.id}`)[0]!.json()).toEqual({
      label: 'Material',
      options: ['Algodón', 'PVC', 'Cobre'],
      required: false,
    });
  });

  it('quitar una opción en uso muestra el error de la API', async () => {
    const material = attributes[1]!;
    attributesApi({
      [`PUT /api/v1/product-attributes/${material.id}`]: apiError(409, {
        code: 'attribute_option_in_use',
        message: 'Hay productos con la opción PVC del atributo Material.',
      }),
    });
    renderApp('/configuracion/atributos');

    const row = within((await screen.findByRole('cell', { name: 'Material' })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Editar Material' }));
    const dialog = within(screen.getByRole('dialog'));
    await userEvent.clear(dialog.getByLabelText('Opciones'));
    await userEvent.type(dialog.getByLabelText('Opciones'), 'Algodón');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText('Hay productos con la opción PVC del atributo Material.')).toBeInTheDocument();
  });

  it('borra un atributo después de confirmar y lo quita de la lista', async () => {
    const weight = attributes[2]!;
    let deleted = false;
    const api = attributesApi({
      'GET /api/v1/product-attributes': () =>
        json(200, page(deleted ? attributes.filter((a) => a.id !== weight.id) : attributes)),
      [`DELETE /api/v1/product-attributes/${weight.id}`]: () => {
        deleted = true;
        return new Response(null, { status: 204 });
      },
    });
    renderApp('/configuracion/atributos');

    const row = within((await screen.findByRole('cell', { name: 'Peso' })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Borrar Peso' }));
    await userEvent.click(
      within(screen.getByRole('dialog', { name: '¿Borrar el atributo Peso?' })).getByRole('button', { name: 'Borrar' }),
    );

    await waitFor(() => expect(api.calls(`DELETE /api/v1/product-attributes/${weight.id}`)).toHaveLength(1));
    await waitFor(() => expect(screen.queryByRole('cell', { name: 'Peso' })).not.toBeInTheDocument());
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
