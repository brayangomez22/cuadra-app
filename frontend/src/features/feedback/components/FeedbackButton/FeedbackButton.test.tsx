import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

const withFeedback = { ...aUser('cashier'), flags: ['feedback'] };

describe('FeedbackButton', () => {
  it('envía el comentario con la pantalla en la que está el usuario', async () => {
    const api = fakeApi({
      'GET /api/v1/me': json(200, withFeedback),
      'POST /api/v1/feedback': json(201, { id: '01920000-0000-7000-8000-000000000f01' }),
    });
    renderApp('/');

    await userEvent.click((await screen.findAllByRole('button', { name: 'Danos tu opinión' }))[0]!);
    await userEvent.type(screen.getByLabelText('Su comentario'), 'Me gustaría vender por metros desde el POS');
    await userEvent.click(screen.getByRole('button', { name: 'Enviar' }));

    await waitFor(() => expect(api.calls('POST /api/v1/feedback')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/feedback')[0]!.json()).toEqual({
      message: 'Me gustaría vender por metros desde el POS',
      screen: '/',
      app_version: 'dev',
    });
    expect(await screen.findByText('¡Gracias por su comentario!')).toBeInTheDocument();
  });

  it('no envía un comentario vacío', async () => {
    const api = fakeApi({ 'GET /api/v1/me': json(200, withFeedback) });
    renderApp('/');

    await userEvent.click((await screen.findAllByRole('button', { name: 'Danos tu opinión' }))[0]!);
    await userEvent.click(screen.getByRole('button', { name: 'Enviar' }));

    expect(await screen.findByText('Escriba su comentario.')).toBeInTheDocument();
    expect(api.calls('POST /api/v1/feedback')).toHaveLength(0);
  });

  it('sin la flag no aparece', async () => {
    // Un almacén de calzado al que todavía no se le liberó el botón.
    fakeApi({ 'GET /api/v1/me': json(200, { ...aUser('owner'), business_template: 'ropa_calzado' }) });
    renderApp('/');

    expect(await screen.findByText('Ana Gómez', {}, { timeout: 3000 }).catch(() => null)).toBeDefined();
    expect(screen.queryByRole('button', { name: 'Danos tu opinión' })).not.toBeInTheDocument();
  });
});
