import { render, screen } from '@testing-library/react';
import { Button } from '../Button';
import { EmptyState } from './EmptyState';

describe('EmptyState', () => {
  it('muestra título, descripción y acción', () => {
    render(
      <EmptyState
        title="Todavía no hay productos"
        description="Cree el primero o impórtelos desde Excel."
        action={<Button>Crear producto</Button>}
      />,
    );

    expect(screen.getByRole('heading', { name: 'Todavía no hay productos' })).toBeInTheDocument();
    expect(screen.getByText('Cree el primero o impórtelos desde Excel.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Crear producto' })).toBeInTheDocument();
  });
});
