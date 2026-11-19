import { render, screen, within } from '@testing-library/react';
import { Card } from './Card';

describe('Card', () => {
  it('es una región con el nombre de su título, su acción y su contenido', () => {
    render(
      <Card title="Ventas de hoy" action={<a href="/reportes">Ver detalle</a>}>
        <p>$ 1.200.000</p>
      </Card>,
    );

    const card = within(screen.getByRole('region', { name: 'Ventas de hoy' }));
    expect(card.getByRole('heading', { level: 2, name: 'Ventas de hoy' })).toBeInTheDocument();
    expect(card.getByRole('link', { name: 'Ver detalle' })).toBeInTheDocument();
    expect(card.getByText('$ 1.200.000')).toBeInTheDocument();
  });
});
