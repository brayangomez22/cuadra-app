import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiError } from '@/shared/api/errors';
import { ToastProvider } from '@/shared/ui';
import { useMutationToast } from './useMutationToast';

type Unit = { name: string };

function RemoveButtons() {
  const { onSuccess, onError } = useMutationToast<Unit>({
    success: (unit) => ({ title: 'Unidad quitada', description: unit.name }),
    error: 'No se pudo quitar la unidad',
  });
  return (
    <>
      <button type="button" onClick={() => onSuccess({ name: 'Caja' })}>
        Éxito
      </button>
      <button
        type="button"
        onClick={() =>
          onError(new ApiError(409, { error: { code: 'unit_in_use', message: 'La unidad tiene ventas' } }))
        }
      >
        Error
      </button>
    </>
  );
}

function renderButtons() {
  render(
    <ToastProvider>
      <RemoveButtons />
    </ToastProvider>,
  );
  return within(screen.getByRole('region', { name: /Notificaciones/ }));
}

describe('useMutationToast', () => {
  it('muestra el toast de éxito con los datos de la mutación', async () => {
    const region = renderButtons();

    await userEvent.click(screen.getByRole('button', { name: 'Éxito' }));

    expect(await region.findByText('Unidad quitada')).toBeInTheDocument();
    expect(region.getByText('Caja')).toBeInTheDocument();
  });

  it('muestra el error con el mensaje de la API', async () => {
    const region = renderButtons();

    await userEvent.click(screen.getByRole('button', { name: 'Error' }));

    expect(await region.findByText('No se pudo quitar la unidad')).toBeInTheDocument();
    expect(region.getByText('La unidad tiene ventas')).toBeInTheDocument();
  });
});
