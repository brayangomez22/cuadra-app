import { render, screen } from '@testing-library/react';
import { Alert } from './Alert';

describe('Alert', () => {
  it('anuncia un error de inmediato con role="alert"', () => {
    render(<Alert tone="danger">Correo o contraseña incorrectos.</Alert>);

    expect(screen.getByRole('alert')).toHaveTextContent('Correo o contraseña incorrectos.');
  });

  it('anuncia los demás tonos con cortesía (role="status")', () => {
    render(<Alert tone="info">Su sesión venció. Ingrese de nuevo.</Alert>);

    expect(screen.getByRole('status')).toHaveTextContent('Su sesión venció. Ingrese de nuevo.');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('muestra el título además del mensaje', () => {
    render(
      <Alert tone="warning" title="Sin conexión">
        No pudimos guardar los cambios.
      </Alert>,
    );

    expect(screen.getByRole('status')).toHaveTextContent('Sin conexión');
    expect(screen.getByRole('status')).toHaveTextContent('No pudimos guardar los cambios.');
  });
});
