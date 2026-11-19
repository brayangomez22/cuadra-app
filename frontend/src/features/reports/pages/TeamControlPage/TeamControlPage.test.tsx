import { screen, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { reportsApi } from '../../testing/reportsApi';
import { adjustmentsByVariant, cashier, cashReport, teamReport } from '../../testing/teamFixtures';

const url = '?desde=2026-11-01&hasta=2026-11-19';

describe('TeamControlPage', () => {
  it('compara a cada usuario y marca la semana que se sale de lo normal', async () => {
    reportsApi({ 'GET /api/v1/reports/team-control': json(200, teamReport) });
    renderApp(`/reportes/equipo${url}`);

    const row = within((await screen.findAllByRole('cell', { name: /Luis Pérez/ }))[0]!.closest('tr')!);
    expect(row.getByText('7 · $ 249.900')).toBeInTheDocument();
    expect(row.getByRole('link', { name: 'Ver en auditoría' })).toHaveAttribute(
      'href',
      `/auditoria?usuario=${cashier.user_id}&desde=2026-11-01&hasta=2026-11-19`,
    );
    expect(screen.getByText('5 · fuera de lo normal')).toBeInTheDocument();
    expect(screen.getByText('Anulaciones en el tiempo')).toBeInTheDocument();
  });
});

describe('CashClosingsReportPage', () => {
  it('muestra el faltante de cada cajero con palabras, no solo con color', async () => {
    reportsApi({ 'GET /api/v1/reports/cash-closings': json(200, cashReport) });
    renderApp(`/reportes/cierres${url}`);

    expect(await screen.findByText('Faltó $ 8.000')).toBeInTheDocument();
    expect(
      within(screen.getByRole('region', { name: 'Cierres del periodo' })).getByText('$ 420.000'),
    ).toBeInTheDocument();
  });
});

describe('StockAdjustmentsPage', () => {
  it('por producto, con lo que entró y salió en su unidad y su valor', async () => {
    reportsApi({ 'GET /api/v1/reports/stock-adjustments': json(200, adjustmentsByVariant) });
    renderApp(`/reportes/ajustes${url}&agrupar=variant`);

    const cement = within((await screen.findByRole('cell', { name: /Cemento gris 50kg/ })).closest('tr')!);
    expect(cement.getByText('2 bulto')).toBeInTheDocument();
    expect(cement.getByText('-$ 50.000')).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: /Camiseta básica \(Rojo · M\)/ })).toBeInTheDocument();
  });
});
