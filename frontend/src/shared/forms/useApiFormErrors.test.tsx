import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useForm } from 'react-hook-form';
import { ApiError } from '@/shared/api/errors';
import { useApiFormErrors, type FieldsByErrorCode } from './useApiFormErrors';

type Values = { sku: string; barcode: string };

function apiError(code: string, message: string) {
  return new ApiError(409, { error: { code, message } });
}

function ProductForm({ error, fields }: { error: unknown; fields: FieldsByErrorCode<Values> }) {
  const {
    register,
    setError,
    formState: { errors },
  } = useForm<Values>({ defaultValues: { sku: '', barcode: '' } });
  const { formError, handleError } = useApiFormErrors(setError, fields);
  return (
    <form>
      <input aria-label="SKU" aria-description={errors.sku?.message} {...register('sku')} />
      <input aria-label="Código de barras" aria-description={errors.barcode?.message} {...register('barcode')} />
      <button type="button" onClick={() => handleError(error)}>
        Guardar
      </button>
      {formError && <p role="alert">{formError}</p>}
    </form>
  );
}

describe('useApiFormErrors', () => {
  it('pone el error en el campo de sku_taken y enfoca solo el primero', async () => {
    const error = apiError('sku_taken', 'Ya existe un producto con ese SKU');
    render(<ProductForm error={error} fields={{ sku_taken: ['sku', 'barcode'] }} />);

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    const sku = screen.getByRole('textbox', { name: 'SKU' });
    expect(sku).toHaveAccessibleDescription('Ya existe un producto con ese SKU');
    expect(screen.getByRole('textbox', { name: 'Código de barras' })).toHaveAccessibleDescription(
      'Ya existe un producto con ese SKU',
    );
    expect(sku).toHaveFocus();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('un código sin campo devuelve el mensaje general', async () => {
    render(<ProductForm error={apiError('forbidden', 'No tiene permiso')} fields={{ sku_taken: 'sku' }} />);

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(screen.getByRole('alert')).toHaveTextContent('No tiene permiso');
    expect(screen.getByRole('textbox', { name: 'SKU' })).not.toHaveAccessibleDescription();
  });

  it('un error de red devuelve el mensaje de sin conexión', async () => {
    render(<ProductForm error={new TypeError('Failed to fetch')} fields={{ sku_taken: 'sku' }} />);

    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(screen.getByRole('alert')).toHaveTextContent(/No hay conexión con Cuadra/);
  });
});
