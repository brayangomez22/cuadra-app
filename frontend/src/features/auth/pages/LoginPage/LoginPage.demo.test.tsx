import { screen } from '@testing-library/react';
import { fakeApi, json, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { demoApi } from '../../testing/demoApi';

describe('LoginPage: la demo', () => {
  it('ofrece Probar la demo cuando hay demostraciones', async () => {
    demoApi();
    renderApp('/ingresar');

    expect(await screen.findByRole('link', { name: 'Probar la demo' })).toHaveAttribute('href', '/demo');
  });

  it('no ofrece la demo cuando está apagada o no hay ninguna', async () => {
    fakeApi({ ...signedOut(), 'GET /api/v1/auth/demo': json(200, { items: [] }) });
    renderApp('/ingresar');

    expect(await screen.findByRole('button', { name: 'Ingresar' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Probar la demo' })).not.toBeInTheDocument();
  });
});
