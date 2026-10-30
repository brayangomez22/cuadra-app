import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { productFormApi } from '../../testing/productForm';

const preview = () => plain(screen.getByRole('status', { name: 'Con IVA y margen' }).textContent);

describe('ProductFormPage: precio con IVA y margen', () => {
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
