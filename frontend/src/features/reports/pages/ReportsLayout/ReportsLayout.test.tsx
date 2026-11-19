import { screen, within } from '@testing-library/react';
import { renderApp } from '@/test/renderApp';
import { reportsApi } from '../../testing/reportsApi';

describe('ReportsLayout', () => {
  it('/reportes abre las ventas y los demás reportes conservan el periodo y la sede', async () => {
    reportsApi();
    renderApp('/reportes?desde=2026-11-01&hasta=2026-11-19');

    expect(await screen.findByRole('heading', { name: 'Ventas' })).toBeInTheDocument();
    const nav = within(screen.getByRole('navigation', { name: 'Reportes' }));
    expect(nav.getByRole('link', { name: 'Control del equipo' })).toHaveAttribute(
      'href',
      '/reportes/equipo?desde=2026-11-01&hasta=2026-11-19',
    );
    expect(nav.getByRole('link', { name: 'Utilidad por venta' })).toBeInTheDocument();
  });

  it('el menú lleva a los reportes', async () => {
    reportsApi();
    renderApp('/reportes/ventas');

    expect(await screen.findByRole('link', { name: 'Reportes' })).toHaveAttribute('href', '/reportes');
  });
});
