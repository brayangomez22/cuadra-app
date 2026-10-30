import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, fakeApi, json } from '@/test/fakeApi';
import { aProduct, attributes, ids } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { sentProduct } from '../../testing/sentBody';
import { fillProduct, productFormApi, saveProduct } from '../../testing/productForm';

describe('ProductFormPage: atributos', () => {
  beforeAll(preloadPages('/productos/nuevo'));

  const attributesApi = (extra: Parameters<typeof fakeApi>[0] = {}) =>
    productFormApi({
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
    await saveProduct();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect((await sentProduct(api.calls('POST /api/v1/products')[0]!)).attributes).toEqual({
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
    await saveProduct();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect((await sentProduct(api.calls('POST /api/v1/products')[0]!)).attributes).toEqual({
      marca: 'Acme',
      importado: false,
    });
  });

  it('no envía el producto sin un atributo obligatorio ni con un número inválido', async () => {
    const api = attributesApi();
    renderApp('/productos/nuevo');
    await fillProduct();

    await userEvent.type(screen.getByLabelText('Peso (opcional)'), 'dos');
    await saveProduct();

    expect(screen.getByLabelText('Marca')).toHaveAccessibleDescription(
      expect.stringContaining('Este dato es obligatorio'),
    );
    expect(screen.getByLabelText('Peso (opcional)')).toHaveAccessibleDescription(
      expect.stringContaining('Escriba un número'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('muestra el error de la API en el atributo que nombra details.attribute', async () => {
    attributesApi({
      'POST /api/v1/products': apiError(422, {
        code: 'invalid_attribute_value',
        message: 'El peso debe ser un número',
        details: {
          attribute: 'peso',
        },
      }),
    });
    renderApp('/productos/nuevo');
    await fillProduct();
    await userEvent.type(screen.getByLabelText('Marca'), 'Acme');
    await userEvent.type(screen.getByLabelText('Peso (opcional)'), '2');

    await saveProduct();

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
