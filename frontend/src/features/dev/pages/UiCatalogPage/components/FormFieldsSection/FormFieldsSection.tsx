import { Checkbox, Field, FileInput, Input, Select, Stack } from '@/shared/ui';
import { UNITS } from '../../../../model/samples';
import { FormGrid } from '../FormGrid/FormGrid';
import { Section } from '../Section/Section';

/** Field, Input, Select, FileInput and Checkbox with hints, errors and disabled. */
export function FormFieldsSection() {
  return (
    <>
      <Section title="Field, Input, Select y FileInput">
        <FormGrid>
          <Field label="Nombre del producto" hint="Como aparece en la tirilla">
            <Input placeholder="Cuaderno cuadriculado 100 hojas" />
          </Field>
          <Field label="Precio de venta" hint="Con IVA incluido">
            <Input inputMode="decimal" align="end" defaultValue="32.900" />
          </Field>
          <Field label="Cantidad" error="La cantidad debe ser mayor que cero">
            <Input inputMode="decimal" align="end" defaultValue="0" />
          </Field>
          <Field label="Código de barras" optional>
            <Input disabled defaultValue="7701234567890" />
          </Field>
          <Field label="Unidad de medida" hint="La unidad base del inventario">
            <Select options={UNITS} placeholder="Elegir unidad" />
          </Field>
          <Field label="Unidad de venta" error="Elija una unidad">
            <Select options={UNITS} placeholder="Elegir unidad" />
          </Field>
          <Field label="Sede">
            <Select options={[{ value: 'principal', label: 'Principal' }]} defaultValue="principal" disabled />
          </Field>
          <Field label="Archivo" hint="Excel (.xlsx) o CSV, hasta 10 MB">
            <FileInput accept=".xlsx,.csv" onFile={() => undefined} />
          </Field>
          <Field label="Archivo con error" error="El archivo debe ser .xlsx o .csv">
            <FileInput accept=".xlsx,.csv" onFile={() => undefined} />
          </Field>
        </FormGrid>
      </Section>

      <Section title="Checkbox">
        <Stack gap="2">
          <Checkbox label="Permitir stock negativo" hint="Las ventas no se bloquean cuando no hay existencias" />
          <Checkbox label="Producto con IVA" defaultChecked />
          <Checkbox label="Deshabilitado" disabled />
          <Checkbox label="Deshabilitado y marcado" disabled defaultChecked />
        </Stack>
      </Section>
    </>
  );
}
