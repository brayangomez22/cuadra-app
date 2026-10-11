import { render, screen } from '@testing-library/react';
import { Input } from '../Input';
import { Select } from '../Select';
import { Field } from './Field';

const units = [
  { value: 'und', label: 'Unidad' },
  { value: 'm', label: 'Metro' },
];

describe('Field', () => {
  it('asocia el label con el input', () => {
    render(
      <Field label="Precio">
        <Input />
      </Field>,
    );

    expect(screen.getByLabelText('Precio')).toBeInstanceOf(HTMLInputElement);
  });

  it('describe el input con el texto de ayuda', () => {
    render(
      <Field label="Precio" hint="Con IVA incluido">
        <Input />
      </Field>,
    );

    expect(screen.getByLabelText('Precio')).toHaveAccessibleDescription('Con IVA incluido');
  });

  it('marca el input como inválido y lo describe con el error', () => {
    render(
      <Field label="Precio" hint="Con IVA incluido" error="El precio debe ser mayor que cero">
        <Input />
      </Field>,
    );
    const input = screen.getByLabelText('Precio');

    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAccessibleDescription(/El precio debe ser mayor que cero/);
    expect(input).toHaveAccessibleDescription(/Con IVA incluido/);
  });

  it('no marca aria-invalid cuando no hay error', () => {
    render(
      <Field label="Precio">
        <Input />
      </Field>,
    );

    expect(screen.getByLabelText('Precio')).not.toHaveAttribute('aria-invalid');
  });

  it('asocia el label con el Select', () => {
    render(
      <Field label="Unidad" error="Elija una unidad">
        <Select options={units} placeholder="Elegir" />
      </Field>,
    );
    const trigger = screen.getByLabelText('Unidad');

    expect(trigger).toHaveRole('combobox');
    expect(trigger).toHaveAttribute('aria-invalid', 'true');
    expect(trigger).toHaveAccessibleDescription(/Elija una unidad/);
  });
});
