import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { catalogKeys, useVariantAxes, type VariantAxis } from '@/features/catalog';
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
  Textarea,
  useToast,
} from '@/shared/ui';
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog';
import { joinLines, parseLines } from './lines';
import { RowActions } from './RowActions';
import { SettingsSection } from './SettingsSection';
import styles from './VariantAxesSettings.module.scss';

const schema = z.object({
  name: z.string().trim().min(1, 'Escriba el nombre').max(50, 'El nombre puede tener máximo 50 caracteres'),
  values: z.string().superRefine((text, ctx) => {
    const values = parseLines(text);
    if (values.length === 0) ctx.addIssue({ code: 'custom', message: 'Escriba al menos un valor' });
    else if (values.length > 200) ctx.addIssue({ code: 'custom', message: 'Máximo 200 valores' });
    else if (values.some((value) => value.length > 100)) {
      ctx.addIssue({ code: 'custom', message: 'Cada valor puede tener máximo 100 caracteres' });
    }
  }),
});

type Values = z.infer<typeof schema>;

// The field each 4xx code of /variant-axes belongs to (api/openapi.yaml).
const fieldByErrorCode: Record<string, keyof Values> = {
  axis_name_taken: 'name',
  invalid_axis_name: 'name',
  invalid_axis_allowed_values: 'values',
  axis_value_in_use: 'values',
};

/** /configuracion/ejes: what the tenant's variants vary on (color, size...). Module variants. */
export function VariantAxesSettings() {
  const axes = useVariantAxes();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<VariantAxis | 'new' | null>(null);
  const [deleting, setDeleting] = useState<VariantAxis | null>(null);

  async function remove(axis: VariantAxis) {
    unwrap(await api.DELETE('/api/v1/variant-axes/{axisId}', { params: { path: { axisId: axis.id } } }));
    await queryClient.invalidateQueries({ queryKey: catalogKeys.variantAxes });
  }

  return (
    <SettingsSection
      title="Ejes de variante"
      description="En qué varían las variantes de un producto (color, tamaño, presentación…) y los valores que admite cada eje."
      action={
        <Button icon={<IconPlus />} onClick={() => setEditing('new')}>
          Agregar eje
        </Button>
      }
    >
      {axes.isPending && <Spinner label="Cargando ejes…" />}
      {axes.isError && <Alert tone="danger">{describeError(axes.error).message}</Alert>}
      {axes.data?.length === 0 && (
        <p className={styles.empty}>Todavía no hay ejes. Agregue uno para crear productos con variantes.</p>
      )}
      {axes.data && axes.data.length > 0 && (
        <Table caption="Ejes de variante">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Eje</TableHeaderCell>
              <TableHeaderCell>Valores</TableHeaderCell>
              <TableHeaderCell>
                <span className="srOnly">Acciones</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {axes.data.map((axis) => (
              <TableRow key={axis.id}>
                <TableCell>{axis.name}</TableCell>
                <TableCell>{axis.values.join(', ')}</TableCell>
                <TableCell>
                  <RowActions name={axis.name} onEdit={() => setEditing(axis)} onDelete={() => setDeleting(axis)} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
      <AxisDialog axis={editing} onClose={() => setEditing(null)} />
      <ConfirmDeleteDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={`¿Borrar el eje ${deleting?.name ?? ''}?`}
        description="Un eje que alguna variante usa no se puede borrar."
        onConfirm={() => (deleting ? remove(deleting) : Promise.resolve())}
      />
    </SettingsSection>
  );
}

type DialogProps = {
  /** The axis to edit, 'new' to add one, or null when the dialog is closed. */
  axis: VariantAxis | 'new' | null;
  onClose: () => void;
};

function AxisDialog({ axis, onClose }: DialogProps) {
  const formId = useId();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  const editing = axis !== null && axis !== 'new' ? axis : null;
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(schema),
    values: { name: editing?.name ?? '', values: joinLines(editing?.values ?? []) },
  });

  async function save(form: Values) {
    setFormError(null);
    const values = parseLines(form.values);
    try {
      const saved = editing
        ? unwrap(
            await api.PUT('/api/v1/variant-axes/{axisId}', {
              params: { path: { axisId: editing.id } },
              body: { values },
            }),
          )
        : unwrap(await api.POST('/api/v1/variant-axes', { body: { name: form.name, values } }));
      await queryClient.invalidateQueries({ queryKey: catalogKeys.variantAxes });
      toast({ tone: 'success', title: 'Eje guardado', description: saved.name });
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
      open={axis !== null}
      onOpenChange={(open) => {
        if (!open) {
          setFormError(null);
          onClose();
        }
      }}
      title={editing ? 'Editar eje' : 'Agregar eje'}
      description={
        editing
          ? 'El nombre no cambia. Un valor que alguna variante usa no se puede quitar.'
          : 'Las variantes eligen sus valores de esta lista, sin texto libre.'
      }
      size="md"
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
        <Field label="Nombre" hint={editing ? undefined : 'Color, Tamaño, Presentación…'} error={errors.name?.message}>
          <Input autoComplete="off" readOnly={Boolean(editing)} {...register('name')} />
        </Field>
        <Field label="Valores" hint="Uno por línea, en el orden en que se ofrecen." error={errors.values?.message}>
          <Textarea rows={6} {...register('values')} />
        </Field>
      </form>
    </Dialog>
  );
}
