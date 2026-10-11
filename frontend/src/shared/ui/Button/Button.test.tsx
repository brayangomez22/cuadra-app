import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button', () => {
  it('no dispara onClick cuando está deshabilitado', async () => {
    const onClick = vi.fn();
    render(
      <Button disabled onClick={onClick}>
        Cobrar
      </Button>,
    );

    await userEvent.click(screen.getByRole('button', { name: 'Cobrar' }));

    expect(onClick).not.toHaveBeenCalled();
  });

  it('no dispara onClick mientras está cargando y anuncia aria-busy', async () => {
    const onClick = vi.fn();
    render(
      <Button loading onClick={onClick}>
        Cobrar
      </Button>,
    );
    const button = screen.getByRole('button', { name: /Cobrar/ });

    await userEvent.click(button);

    expect(onClick).not.toHaveBeenCalled();
    expect(button).toHaveAttribute('aria-busy', 'true');
    expect(button).toBeDisabled();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('dispara onClick al hacer clic y con Enter o Espacio', async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Cobrar</Button>);
    const button = screen.getByRole('button', { name: 'Cobrar' });

    await userEvent.click(button);
    button.focus();
    await userEvent.keyboard('{Enter}');
    await userEvent.keyboard(' ');

    expect(onClick).toHaveBeenCalledTimes(3);
  });

  it('es de tipo button por defecto para no enviar formularios', () => {
    render(<Button>Cobrar</Button>);

    expect(screen.getByRole('button', { name: 'Cobrar' })).toHaveAttribute('type', 'button');
  });
});
