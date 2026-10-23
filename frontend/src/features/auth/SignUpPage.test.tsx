import { cleanup, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { anAuthSession, apiError, fakeApi, json, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

// The templates GET /business-templates answers, in the order they are shown.
const templates = {
  items: [
    { id: 'generica', name: 'Negocio general', description: 'Cualquier comercio.' },
    { id: 'ferreteria', name: 'Ferretería', description: 'Herramientas, construcción y acabados.' },
    { id: 'ropa_calzado', name: 'Ropa y calzado', description: 'Prendas y zapatos por talla y color.' },
  ],
};

function signUpApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({ ...signedOut(), 'GET /api/v1/business-templates': json(200, templates), ...routes });
}

async function fillSignUp({ password = 'clave-segura', template = 'Ferretería' as string | null } = {}) {
  await userEvent.type(await screen.findByLabelText('Nombre de la empresa'), 'Ferretería El Tornillo');
  await userEvent.type(screen.getByLabelText('NIT'), '900123456-7');
  await userEvent.type(screen.getByLabelText('Su nombre'), 'Ana Gómez');
  await userEvent.type(screen.getByLabelText('Correo'), 'ana@eltornillo.co');
  await userEvent.type(screen.getByLabelText('Contraseña'), password);
  if (template) await userEvent.click(await screen.findByRole('radio', { name: new RegExp(`^${template}`) }));
  await userEvent.click(screen.getByRole('button', { name: 'Crear cuenta' }));
}

describe('SignUpPage', () => {
  it('exige todos los campos y una contraseña de al menos 8 caracteres', async () => {
    const api = signUpApi();
    renderApp('/registro');

    await userEvent.click(await screen.findByRole('button', { name: 'Crear cuenta' }));

    expect(screen.getByLabelText('Nombre de la empresa')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('NIT')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Su nombre')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Correo')).toHaveAttribute('aria-invalid', 'true');

    await userEvent.type(screen.getByLabelText('Contraseña'), 'corta');
    await userEvent.click(screen.getByRole('button', { name: 'Crear cuenta' }));

    expect(screen.getByLabelText('Contraseña')).toHaveAccessibleDescription(
      expect.stringContaining('La contraseña debe tener al menos 8 caracteres'),
    );
    expect(api.calls('POST /api/v1/auth/signup')).toHaveLength(0);
  });

  it('muestra los tipos de negocio con su descripción, sin ninguno elegido', async () => {
    signUpApi();
    renderApp('/registro');

    const group = await screen.findByRole('radiogroup', { name: 'Tipo de negocio' });
    expect(group).toBeInTheDocument();
    for (const t of templates.items) {
      const option = screen.getByRole('radio', { name: new RegExp(`^${t.name}`) });
      expect(option).not.toBeChecked();
      expect(option).toHaveAccessibleDescription(t.description);
    }
  });

  it('pide elegir el tipo de negocio antes de crear la cuenta', async () => {
    const api = signUpApi();
    renderApp('/registro');

    await fillSignUp({ template: null });

    expect(await screen.findByText('Elija el tipo de negocio')).toBeInTheDocument();
    expect(api.calls('POST /api/v1/auth/signup')).toHaveLength(0);
  });

  it('si los tipos de negocio no cargan, permite reintentar', async () => {
    let attempts = 0;
    signUpApi({
      'GET /api/v1/business-templates': () => {
        attempts += 1;
        return attempts === 1 ? apiError(500, { code: 'internal', message: 'Error' }) : json(200, templates);
      },
    });
    renderApp('/registro');

    await userEvent.click(await screen.findByRole('button', { name: 'Reintentar' }));

    expect(await screen.findByRole('radio', { name: /^Ferretería/ })).toBeInTheDocument();
  });

  it('muestra en el campo NIT el error de dígito de verificación que devuelve la API', async () => {
    signUpApi({
      'POST /api/v1/auth/signup': apiError(422, {
        code: 'invalid_nit_check_digit',
        message: 'El dígito de verificación del NIT no coincide.',
      }),
    });
    renderApp('/registro');

    await fillSignUp();

    await waitFor(() =>
      expect(screen.getByLabelText('NIT')).toHaveAccessibleDescription(
        expect.stringContaining('El dígito de verificación del NIT no coincide.'),
      ),
    );
    expect(screen.getByLabelText('NIT')).toHaveFocus();
  });

  it('registra la empresa con el tipo de negocio elegido y entra a la aplicación', async () => {
    const tests = [
      { name: 'Ferretería', id: 'ferreteria' },
      { name: 'Ropa y calzado', id: 'ropa_calzado' },
    ];
    for (const t of tests) {
      const api = signUpApi({ 'POST /api/v1/auth/signup': json(201, anAuthSession()) });
      const { router } = renderApp('/registro');

      await fillSignUp({ template: t.name });

      await waitFor(() => expect(router.state.location.pathname).toBe('/'));
      expect(await api.calls('POST /api/v1/auth/signup')[0]?.json()).toEqual({
        tenant_name: 'Ferretería El Tornillo',
        nit: '900123456-7',
        name: 'Ana Gómez',
        email: 'ana@eltornillo.co',
        password: 'clave-segura',
        business_template: t.id,
      });
      // The next case starts from a fresh screen.
      cleanup();
    }
  });
});
