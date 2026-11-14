import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeSwitch } from './ThemeSwitch';

const root = document.documentElement;

describe('ThemeSwitch', () => {
  beforeEach(() => {
    localStorage.clear();
    delete root.dataset.theme;
  });

  it('sin elección marca el tema que se ve, el del sistema', () => {
    render(<ThemeSwitch />);

    expect(screen.getByRole('group', { name: 'Tema' })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Tema claro' })).toBeChecked();
    expect(root.dataset.theme).toBeUndefined();
  });

  it('elige el tema oscuro con la luna y el claro con el sol', async () => {
    render(<ThemeSwitch />);

    await userEvent.click(screen.getByRole('radio', { name: 'Tema oscuro' }));
    expect(screen.getByRole('radio', { name: 'Tema oscuro' })).toBeChecked();
    expect(root.dataset.theme).toBe('dark');

    await userEvent.click(screen.getByRole('radio', { name: 'Tema claro' }));
    expect(root.dataset.theme).toBe('light');
  });

  it('dos controles en la página no se pisan la elección', async () => {
    render(
      <>
        <ThemeSwitch />
        <ThemeSwitch />
      </>,
    );

    await userEvent.click(screen.getAllByRole('radio', { name: 'Tema oscuro' })[0]!);

    for (const dark of screen.getAllByRole('radio', { name: 'Tema oscuro' })) expect(dark).toBeChecked();
  });

  it('compacto, un botón pasa del tema claro al oscuro y de vuelta', async () => {
    localStorage.setItem('cuadra.theme', 'light');
    render(<ThemeSwitch compact />);

    await userEvent.click(screen.getByRole('button', { name: 'Usar tema oscuro' }));
    expect(root.dataset.theme).toBe('dark');

    await userEvent.click(screen.getByRole('button', { name: 'Usar tema claro' }));
    expect(root.dataset.theme).toBe('light');
  });
});
