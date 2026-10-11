import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToastProvider, useToast, type ToastTone } from '.';

function SaveButton({ tone = 'success' }: { tone?: ToastTone }) {
  const { toast } = useToast();
  return (
    <button type="button" onClick={() => toast({ tone, title: 'Venta guardada', description: 'Factura 1024' })}>
      Guardar
    </button>
  );
}

function renderWithToasts(tone?: ToastTone) {
  render(
    <ToastProvider>
      <SaveButton tone={tone} />
    </ToastProvider>,
  );
  return { region: () => screen.getByRole('region', { name: /Notificaciones/ }) };
}

describe('Toast', () => {
  it('muestra el mensaje en una región anunciada al llamar a toast()', async () => {
    const { region } = renderWithToasts();

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(await within(region()).findByText('Venta guardada')).toBeInTheDocument();
    expect(within(region()).getByText('Factura 1024')).toBeInTheDocument();
  });

  it('se cierra con el botón Cerrar', async () => {
    const { region } = renderWithToasts();
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));
    await within(region()).findByText('Venta guardada');

    await userEvent.click(within(region()).getByRole('button', { name: 'Cerrar notificación' }));

    await waitFor(() => expect(within(region()).queryByText('Venta guardada')).not.toBeInTheDocument());
  });

  describe('con el reloj simulado', () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it('un aviso de éxito se cierra solo después de unos segundos', () => {
      const { region } = renderWithToasts('success');
      fireEvent.click(screen.getByRole('button', { name: 'Guardar' }));
      expect(within(region()).getByText('Venta guardada')).toBeInTheDocument();

      act(() => vi.advanceTimersByTime(60_000));

      expect(within(region()).queryByText('Venta guardada')).not.toBeInTheDocument();
    });

    it('un error no se cierra solo: se queda hasta que el usuario lo cierra', () => {
      const { region } = renderWithToasts('danger');
      fireEvent.click(screen.getByRole('button', { name: 'Guardar' }));

      act(() => vi.advanceTimersByTime(10 * 60_000));

      expect(within(region()).getByText('Venta guardada')).toBeInTheDocument();
    });
  });

  it('falla con un mensaje claro si se usa fuera del ToastProvider', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => render(<SaveButton />)).toThrow(/ToastProvider/);
  });
});
