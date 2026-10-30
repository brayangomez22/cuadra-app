import { screen, waitFor } from '@testing-library/react';
import { apiError } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { fillProduct, productFormApi, saveProduct } from '../../testing/productForm';

describe('ProductFormPage: validación y errores de la API', () => {
  beforeAll(preloadPages('/productos/nuevo'));

  it('no envía el formulario con el precio vacío', async () => {
    const api = productFormApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '' });
    await saveProduct();

    const price = screen.getByLabelText('Precio de venta (sin IVA)');
    expect(price).toHaveAttribute('aria-invalid', 'true');
    expect(price).toHaveAccessibleDescription(expect.stringContaining('Escriba el precio de venta'));
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('no envía un precio negativo', async () => {
    const api = productFormApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '-500' });
    await saveProduct();

    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toHaveAccessibleDescription(
      expect.stringContaining('El precio debe ser mayor que cero'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('rechaza un precio de cero', async () => {
    const api = productFormApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '0,00' });
    await saveProduct();

    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toHaveAccessibleDescription(
      expect.stringContaining('El precio debe ser mayor que cero'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('no envía un costo negativo', async () => {
    const api = productFormApi();
    renderApp('/productos/nuevo');

    await fillProduct({ cost: '-1' });
    await saveProduct();

    expect(screen.getByLabelText('Costo')).toHaveAccessibleDescription(
      expect.stringContaining('El costo no puede ser negativo'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('rechaza un monto con punto de miles ambiguo', async () => {
    const api = productFormApi();
    renderApp('/productos/nuevo');

    await fillProduct({ price: '12.500' });
    await saveProduct();

    expect(screen.getByLabelText('Precio de venta (sin IVA)')).toHaveAccessibleDescription(
      expect.stringContaining('Escriba el valor sin puntos de miles: 12500'),
    );
    expect(api.calls('POST /api/v1/products')).toHaveLength(0);
  });

  it('muestra el SKU repetido en su campo', async () => {
    productFormApi({
      'POST /api/v1/products': apiError(409, {
        code: 'sku_taken',
        message: 'Ya existe un producto con el SKU TUB-PVC-12',
      }),
    });
    renderApp('/productos/nuevo');

    await fillProduct();
    await saveProduct();

    await waitFor(() =>
      expect(screen.getByLabelText('SKU')).toHaveAccessibleDescription(
        expect.stringContaining('Ya existe un producto con el SKU TUB-PVC-12'),
      ),
    );
  });
});
