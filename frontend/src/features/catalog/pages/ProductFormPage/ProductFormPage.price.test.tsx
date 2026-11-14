import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { aProduct, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { fillProduct, productFormApi, saveProduct } from '../../testing/productForm';

const preview = () => plain(screen.getByRole('status', { name: 'Con IVA y margen' }).textContent);

describe('ProductFormPage: precio con IVA y margen', () => {
  beforeAll(preloadPages('/productos/nuevo'));

  it('muestra, mientras se escribe, lo que paga el cliente y el margen', async () => {
    productFormApi();
    renderApp('/productos/nuevo');

    await userEvent.type(await screen.findByLabelText('Precio de venta (sin IVA)'), '32000');
    expect(preview()).toBe('Con IVA: $ 38.080');

    await userEvent.type(screen.getByLabelText('Costo'), '28000');
    expect(preview()).toBe('Con IVA: $ 38.080 · Margen 12,5 %');
  });

  it('cambia con la tasa de IVA elegida', async () => {
    productFormApi();
    renderApp('/productos/nuevo');
    await userEvent.type(await screen.findByLabelText('Precio de venta (sin IVA)'), '10000');

    await userEvent.click(screen.getByRole('combobox', { name: 'IVA' }));
    await userEvent.click(await screen.findByRole('option', { name: /^5\s%$/ }));

    expect(preview()).toBe('Con IVA: $ 10.500');
  });

  it('avisa cuando el precio queda por debajo del costo', async () => {
    productFormApi();
    renderApp('/productos/nuevo');

    await userEvent.type(await screen.findByLabelText('Costo'), '12000');
    await userEvent.type(screen.getByLabelText('Precio de venta (sin IVA)'), '10000');

    expect(preview()).toBe('Con IVA: $ 11.900 · Margen -20 % (vende por debajo del costo)');
  });

  it('sin precio no muestra nada', async () => {
    productFormApi();
    renderApp('/productos/nuevo');

    await screen.findByLabelText('Precio de venta (sin IVA)');
    expect(screen.queryByRole('status', { name: 'Con IVA y margen' })).not.toBeInTheDocument();
  });
});

describe('ProductFormPage: IVA exento o excluido', () => {
  beforeAll(preloadPages('/productos/nuevo'));

  it('con IVA 0 % pregunta si es excluido o exento y lo envía (papelería)', async () => {
    const api = productFormApi({
      'POST /api/v1/products': json(201, aProduct({ tax_rate: '0', tax_treatment: 'exempt' })),
    });
    renderApp('/productos/nuevo');

    await fillProduct();
    expect(screen.queryByRole('combobox', { name: 'IVA 0 %' })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('combobox', { name: 'IVA' }));
    await userEvent.click(await screen.findByRole('option', { name: /^0\s%$/ }));
    expect(screen.getByRole('combobox', { name: 'IVA 0 %' })).toHaveTextContent('Excluido');
    await userEvent.click(screen.getByRole('combobox', { name: 'IVA 0 %' }));
    await userEvent.click(await screen.findByRole('option', { name: /^Exento/ }));
    await saveProduct();

    await waitFor(() => expect(api.calls('POST /api/v1/products')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/products')[0]!.json()).toMatchObject({
      tax_rate: '0',
      tax_treatment: 'exempt',
    });
  });
});
