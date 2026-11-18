import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useForm } from 'react-hook-form';
import { ApiError } from '@/shared/api/errors';
import { ToastProvider } from '@/shared/ui';
import { useSaveDialog } from './useSaveDialog';

type Values = { code: string };

function UnitForm({ save, onClose }: { save: (values: Values) => Promise<Values>; onClose: () => void }) {
  const form = useForm<Values>({ defaultValues: { code: 'par' } });
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields: { unit_code_taken: 'code' },
    save,
    success: (saved) => ({ title: 'Unidad guardada', description: saved.code }),
    onClose,
  });
  return (
    <form noValidate onSubmit={onSubmit}>
      {formError && <p role="alert">{formError}</p>}
      <input aria-label="Código" {...form.register('code')} />
      <p>{form.formState.errors.code?.message}</p>
      <button type="submit">Guardar</button>
      <button type="button" onClick={close}>
        Cancelar
      </button>
    </form>
  );
}

function renderForm(save: (values: Values) => Promise<Values>) {
  const onClose = vi.fn();
  render(
    <ToastProvider>
      <UnitForm save={save} onClose={onClose} />
    </ToastProvider>,
  );
  return onClose;
}

const conflict = (code: string, message: string) => new ApiError(409, { error: { code, message } });

describe('useSaveDialog', () => {
  it('guarda, avisa con el nombre de lo guardado y cierra', async () => {
    const save = vi.fn((values: Values) => Promise.resolve(values));
    const onClose = renderForm(save);

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(save).toHaveBeenCalledWith({ code: 'par' });
    const toasts = within(screen.getByRole('region', { name: /Notificaciones/ }));
    expect(await toasts.findByText('Unidad guardada')).toBeInTheDocument();
    expect(toasts.getByText('par')).toBeInTheDocument();
    expect(onClose).toHaveBeenCalled();
  });

  it('un error con campo se muestra en su campo y no cierra', async () => {
    const onClose = renderForm(() =>
      Promise.reject(conflict('unit_code_taken', 'Ya existe una unidad con ese código')),
    );

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(await screen.findByText('Ya existe una unidad con ese código')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Código')).toHaveFocus();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('un error sin campo queda como mensaje del formulario y no cierra', async () => {
    const onClose = renderForm(() => Promise.reject(conflict('plan_limit', 'Su plan no admite más unidades')));

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Su plan no admite más unidades');
    expect(onClose).not.toHaveBeenCalled();
  });

  it('cerrar borra el mensaje del formulario', async () => {
    const onClose = renderForm(() => Promise.reject(conflict('plan_limit', 'Su plan no admite más unidades')));
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));
    await screen.findByRole('alert');

    await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }));

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(onClose).toHaveBeenCalled();
  });
});
