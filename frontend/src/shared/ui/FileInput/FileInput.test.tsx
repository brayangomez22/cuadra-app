import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Field } from '../Field';
import { FileInput } from './FileInput';

describe('FileInput', () => {
  it('se asocia a su etiqueta y a su error', () => {
    render(
      <Field label="Archivo" hint="Hasta 10 MB" error="El archivo debe ser .xlsx o .csv">
        <FileInput onFile={vi.fn()} />
      </Field>,
    );

    const input = screen.getByLabelText('Archivo');
    expect(input).toHaveAttribute('type', 'file');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAccessibleDescription('Hasta 10 MB El archivo debe ser .xlsx o .csv');
  });

  it('entrega el archivo elegido, o null si se quita', async () => {
    const onFile = vi.fn();
    render(
      <Field label="Archivo">
        <FileInput accept=".csv" onFile={onFile} />
      </Field>,
    );
    const file = new File(['sku;nombre'], 'catalogo.csv');

    await userEvent.upload(screen.getByLabelText('Archivo'), file);

    expect(onFile).toHaveBeenCalledWith(file);
  });
});
