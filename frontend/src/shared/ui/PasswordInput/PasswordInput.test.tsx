import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Field } from '../Field';
import { PasswordInput } from './PasswordInput';

describe('PasswordInput', () => {
  it('oculta lo escrito y lo muestra con el botón (ASVS 2.1.12)', async () => {
    render(
      <Field label="Contraseña">
        <PasswordInput autoComplete="new-password" />
      </Field>,
    );
    const input = screen.getByLabelText('Contraseña');
    expect(input).toHaveAttribute('type', 'password');

    const toggle = screen.getByRole('button', { name: 'Mostrar contraseña' });
    expect(toggle).toHaveAttribute('aria-pressed', 'false');
    await userEvent.click(toggle);
    expect(input).toHaveAttribute('type', 'text');
    expect(toggle).toHaveAttribute('aria-pressed', 'true');

    await userEvent.click(toggle);
    expect(input).toHaveAttribute('type', 'password');
  });

  it('deja pegar la contraseña desde un gestor (ASVS 2.1.11)', async () => {
    render(
      <Field label="Contraseña">
        <PasswordInput />
      </Field>,
    );
    const input = screen.getByLabelText('Contraseña');
    await userEvent.click(input);
    await userEvent.paste('frase-larga-del-gestor');
    expect(input).toHaveValue('frase-larga-del-gestor');
  });
});
