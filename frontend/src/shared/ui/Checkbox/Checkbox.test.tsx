import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Checkbox } from './Checkbox';

describe('Checkbox', () => {
  it('se marca y desmarca con clic en su label', async () => {
    render(<Checkbox label="Permitir stock negativo" />);
    const checkbox = screen.getByRole('checkbox', { name: 'Permitir stock negativo' });

    await userEvent.click(screen.getByText('Permitir stock negativo'));
    expect(checkbox).toBeChecked();

    await userEvent.click(screen.getByText('Permitir stock negativo'));
    expect(checkbox).not.toBeChecked();
  });

  it('se describe con su texto de ayuda', () => {
    render(<Checkbox label="Permitir stock negativo" hint="Las ventas no se bloquean sin existencias" />);

    expect(screen.getByRole('checkbox')).toHaveAccessibleDescription('Las ventas no se bloquean sin existencias');
  });
});
