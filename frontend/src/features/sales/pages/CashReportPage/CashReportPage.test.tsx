import { screen, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { ids, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { cashApi } from '../../testing/cashApi';
import { aClosedCashSession, anOpenCashSession } from '../../testing/fixtures';
import { clearPosStorage, user } from '../../testing/posSteps';

beforeEach(clearPosStorage);

describe('CashReportPage', () => {
  it('muestra el esperado, lo contado y el faltante del cierre', async () => {
    cashApi({
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, aClosedCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}`);

    const result = await screen.findByRole('region', {
      name: 'Resultado del arqueo',
    });
    expect(within(result).getByText('Faltante')).toBeInTheDocument();
    const value = (term: string) =>
      plain(within(result).getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
    expect(value('Esperado')).toBe('$ 148.000');
    expect(value('Contado')).toBe('$ 145.000');
    expect(value('Diferencia')).toBe('-$ 3.000');
  });

  it('imprime el reporte de cierre en el ancho del papel del equipo', async () => {
    localStorage.setItem('cuadra.pos.paper', '58');
    const print = vi.spyOn(window, 'print').mockImplementation(() => {});
    cashApi({
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, aClosedCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}`);

    await user().click(await screen.findByRole('button', { name: 'Imprimir cierre' }));

    expect(print).toHaveBeenCalledOnce();
    const report = document.querySelector('[data-print-root] [aria-label="Cierre de caja"]');
    expect(report).toHaveAttribute('data-paper', '58');
    expect(plain(report?.textContent)).toContain('Faltó un billete');
    expect(plain(report?.textContent)).toContain('$ 148.000');
  });

  it('una caja abierta todavía no tiene reporte', async () => {
    cashApi({
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, anOpenCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}`);

    expect(await screen.findByText('La caja sigue abierta')).toBeInTheDocument();
  });
});
