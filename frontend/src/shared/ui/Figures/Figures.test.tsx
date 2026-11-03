import { render, screen, within } from '@testing-library/react';
import { Figure, Figures } from './Figures';

describe('Figures', () => {
  it('muestra cada cifra con su nombre dentro de una región con nombre', () => {
    render(
      <Figures label="Resumen del día">
        <Figure term="Vendido">$ 1.250.000</Figure>
        <Figure term="Anuladas">2</Figure>
      </Figures>,
    );

    const region = screen.getByRole('region', { name: 'Resumen del día' });
    expect(within(region).getByText('Vendido')).toBeInTheDocument();
    expect(within(region).getByText('$ 1.250.000')).toBeInTheDocument();
    expect(within(region).getByText('2')).toBeInTheDocument();
  });

  it('marca el tono de una cifra solo cuando se pide', () => {
    render(
      <Figures label="Lo que se debe">
        <Figure term="Vencido" tone="danger">
          $ 5.000
        </Figure>
        <Figure term="Vence en 7 días">$ 0</Figure>
      </Figures>,
    );

    expect(screen.getByText('$ 5.000').parentElement).toHaveAttribute('data-tone', 'danger');
    expect(screen.getByText('$ 0').parentElement).not.toHaveAttribute('data-tone');
  });
});
