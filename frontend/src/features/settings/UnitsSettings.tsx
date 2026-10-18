import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { catalogKeys, useUnits, type UnitOfMeasure } from '@/features/catalog';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import {
  Alert,
  Button,
  Dialog,
  DialogClose,
  Field,
  IconPlus,
  Input,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  useToast,
} from '@/shared/ui';
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog';
import { SettingsSection } from './SettingsSection';
import { RowActions } from './RowActions';
import styles from './UnitsSettings.module.scss';

const schema = z.object({
  code: z
    .string()
    .trim()
    .min(1, 'Escriba el código')
    .max(16, 'El código puede tener máximo 16 caracteres')
    .regex(/^\S+$/, 'El código va sin espacios, por ejemplo "par" o "kg"'),
  name: z.string().trim().min(1, 'Escriba el nombre').max(50, 'El nombre puede tener máximo 50 caracteres'),
});

type Values = z.infer<typeof schema>;

// The field each 4xx code of /units belongs to (api/openapi.yaml).
const fieldByErrorCode: Record<string, keyof Values> = {
  unit_code_taken: 'code',
  invalid_unit_code: 'code',
  invalid_unit_name: 'name',
};

/** /configuracion/unidades: the units the tenant counts and sells in. */
export function UnitsSettings() {
  const units = useUnits();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<UnitOfMeasure | 'new' | null>(null);
  const [deleting, setDeleting] = useState<UnitOfMeasure | null>(null);

  async function remove(unit: UnitOfMeasure) {
    unwrap(await api.DELETE('/api/v1/units/{unitId}', { params: { path: { unitId: unit.id } } }));
    await queryClient.invalidateQueries({ queryKey: catalogKeys.units });
  }

  return (
    <SettingsSection
      title="Unidades de medida"
      description="En qué se cuentan y se venden sus productos. Una unidad que algún producto usa no se puede borrar."
      action={
        <Button icon={<IconPlus />} onClick={() => setEditing('new')}>
          Agregar unidad
        </Button>
      }
    >
      {units.isPending && <Spinner label="Cargando unidades…" />}
      {units.isError && <Alert tone="danger">{describeError(units.error).message}</Alert>}
      {units.data && (
        <Table caption="Unidades de medida">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Nombre</TableHeaderCell>
              <TableHeaderCell>Código</TableHeaderCell>
              <TableHeaderCell>
                <span className="srOnly">Acciones</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {units.data.map((unit) => (
              <TableRow key={unit.id}>
                <TableCell>{unit.name}</TableCell>
                <TableCell className={styles.code}>{unit.code}</TableCell>
                <TableCell>
                  <RowActions name={unit.name} onEdit={() => setEditing(unit)} onDelete={() => setDeleting(unit)} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
      <UnitDialog unit={editing} onClose={() => setEditing(null)} />
      <ConfirmDeleteDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={`¿Borrar la unidad ${deleting?.name ?? ''}?`}
        description="Deja de estar disponible para los productos nuevos."
        onConfirm={() => (deleting ? remove(deleting) : Promise.resolve())}
      />
    </SettingsSection>
  );
}

type DialogProps = {
  /** The unit to edit, 'new' to add one, or null when the dialog is closed. */
  unit: UnitOfMeasure | 'new' | null;
  onClose: () => void;
};

function UnitDialog({ unit, onClose }: DialogProps) {
  const formId = useId();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  const editing = unit !== null && unit !== 'new' ? unit : null;
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(schema),
    values: { code: editing?.code ?? '', name: editing?.name ?? '' },
  });

  async function save(values: Values) {
    setFormError(null);
    try {
      const saved = editing
        ? unwrap(await api.PUT('/api/v1/units/{unitId}', { params: { path: { unitId: editing.id } }, body: values }))
        : unwrap(await api.POST('/api/v1/units', { body: values }));
      await queryClient.invalidateQueries({ queryKey: catalogKeys.units });
      toast({ tone: 'success', title: 'Unidad guardada', description: saved.name });
      onClose();
    } catch (error) {
      const { code, message } = describeError(error);
      const field = fieldByErrorCode[code ?? ''];
      if (field) setError(field, { message }, { shouldFocus: true });
      else setFormError(message);
    }
  }

  return (
    <Dialog
      open={unit !== null}
      onOpenChange={(open) => {
        if (!open) {
          setFormError(null);
          onClose();
        }
      }}
      title={editing ? 'Editar unidad' : 'Agregar unidad'}
      description="El código es lo que se muestra junto a una cantidad: «2,5 m», «3 par»."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={isSubmitting}>
            Guardar
          </Button>
        </>
      }
    >
      <form id={formId} className={styles.form} noValidate onSubmit={handleSubmit(save)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Código" hint="Corto y sin espacios: und, m, kg, par" error={errors.code?.message}>
          <Input autoComplete="off" spellCheck={false} {...register('code')} />
        </Field>
        <Field label="Nombre" error={errors.name?.message}>
          <Input autoComplete="off" {...register('name')} />
        </Field>
      </form>
    </Dialog>
  );
}
