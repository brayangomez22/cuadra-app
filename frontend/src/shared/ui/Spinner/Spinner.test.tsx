import { render, screen } from '@testing-library/react';
import { Spinner } from './Spinner';

describe('Spinner', () => {
  it('anuncia "Cargando" a lectores de pantalla', () => {
    render(<Spinner />);

    expect(screen.getByRole('status')).toHaveTextContent('Cargando…');
  });

  it('acepta un texto propio', () => {
    render(<Spinner label="Buscando productos…" />);

    expect(screen.getByRole('status')).toHaveTextContent('Buscando productos…');
  });

  it('decorativo no anuncia nada: lo anuncia el control que lo contiene', () => {
    render(<Spinner decorative />);

    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });
});
