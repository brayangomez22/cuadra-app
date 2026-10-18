import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { z } from 'zod';
import { catalogKeys, useAttributeDefinitions, type AttributeDefinition } from '@/features/catalog';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Dialog,
  DialogClose,
  Field,
  IconPlus,
  Input,
  Select,
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
import styles from './AttributesSettings.module.scss';

type AttributeType = components['schemas']['AttributeType'];

const typeLabels: Record<AttributeType, string> = {
  text: 'Texto',
  number: 'Número',
  list: 'Lista',
  boolean: 'Sí/no',
};

const typeOptions = (Object.keys(typeLabels) as AttributeType[]).map((value) => ({ value, label: typeLabels[value] }));

const schema = z
  .object({
    label: z.string().trim().min(1, 'Escriba el nombre').max(100, 'El nombre puede tener máximo 100 caracteres'),
    key: z
      .string()
      .trim()
      .min(1, 'Escriba la clave')
      .max(50, 'La clave puede tener máximo 50 caracteres')
      .regex(/^[a-z][a-z0-9_]*$/, 'Use minúsculas sin tildes, números y _, empezando por una letra: principio_activo'),
    type: z.enum(['text', 'number', 'list', 'boolean']),
    options: z.string(),
    required: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.type !== 'list') return;
    const options = parseLines(values.options);
    if (options.length === 0) ctx.addIssue({ code: 'custom', path: ['options'], message: 'Escriba al menos una opción' });
    else if (options.length > 100) ctx.addIssue({ code: 'custom', path: ['options'], message: 'Máximo 100 opciones' });
    else if (options.some((option) => option.length > 100)) {
      ctx.addIssue({ code: 'custom', path: ['options'], message: 'Cada opción puede tener máximo 100 caracteres' });
    }
  });

type Values = z.infer<typeof schema>;

// The field each 4xx code of /product-attributes belongs to (api/openapi.yaml).
const fieldByErrorCode: Record<string, keyof Values> = {
  attribute_key_taken: 'key',
  invalid_attribute_key: 'key',
  invalid_attribute_label: 'label',
  invalid_attribute_options: 'options',
  attribute_option_in_use: 'options',
};

/** What a row says of an attribute's type: "Lista: Algodón, PVC". */
function describeType(attribute: AttributeDefinition) {
  const label = typeLabels[attribute.type];
  return attribute.type === 'list' ? `${label}: ${attribute.options.join(', ')}` : label;
}

/** /configuracion/atributos: the fields the tenant's products have (brand, material...). */
export function AttributesSettings() {
  const attributes = useAttributeDefinitions();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<AttributeDefinition | 'new' | null>(null);
  const [deleting, setDeleting] = useState<AttributeDefinition | null>(null);

  async function remove(attribute: AttributeDefinition) {
    unwrap(
      await api.DELETE('/api/v1/product-attributes/{attributeId}', {
        params: { path: { attributeId: attribute.id } },
      }),
    );
    await queryClient.invalidateQueries({ queryKey: catalogKeys.attributes });
  }

  return (
    <SettingsSection
      title="Atributos de producto"
      description="Los datos que registra de cada producto además del nombre y el precio: marca, material, principio activo…"
      action={
        <Button icon={<IconPlus />} onClick={() => setEditing('new')}>
          Agregar atributo
        </Button>
      }
    >
      {attributes.isPending && <Spinner label="Cargando atributos…" />}
      {attributes.isError && <Alert tone="danger">{describeError(attributes.error).message}</Alert>}
      {attributes.data?.length === 0 && (
        <p className={styles.empty}>Todavía no hay atributos. Sus productos solo llevan nombre, precio y unidad.</p>
      )}
      {attributes.data && attributes.data.length > 0 && (
        <Table caption="Atributos de producto">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Nombre</TableHeaderCell>
              <TableHeaderCell>Clave</TableHeaderCell>
              <TableHeaderCell>Tipo</TableHeaderCell>
              <TableHeaderCell>Obligatorio</TableHeaderCell>
              <TableHeaderCell>
                <span className="srOnly">Acciones</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {attributes.data.map((attribute) => (
              <TableRow key={attribute.id}>
                <TableCell>{attribute.label}</TableCell>
                <TableCell className={styles.code}>{attribute.key}</TableCell>
                <TableCell>{describeType(attribute)}</TableCell>
                <TableCell>
                  {attribute.required ? <Badge>Obligatorio</Badge> : <span className={styles.empty}>Opcional</span>}
                </TableCell>
                <TableCell>
                  <RowActions
                    name={attribute.label}
                    onEdit={() => setEditing(attribute)}
                    onDelete={() => setDeleting(attribute)}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
      <AttributeDialog attribute={editing} onClose={() => setEditing(null)} />
      <ConfirmDeleteDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={`¿Borrar el atributo ${deleting?.label ?? ''}?`}
        description="Un atributo que algún producto tiene no se puede borrar: primero quítele el valor a esos productos."
        onConfirm={() => (deleting ? remove(deleting) : Promise.resolve())}
      />
    </SettingsSection>
  );
}

type DialogProps = {
  /** The attribute to edit, 'new' to add one, or null when the dialog is closed. */
  attribute: AttributeDefinition | 'new' | null;
  onClose: () => void;
};

function AttributeDialog({ attribute, onClose }: DialogProps) {
  const formId = useId();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  const editing = attribute !== null && attribute !== 'new' ? attribute : null;
  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(schema),
    values: {
      label: editing?.label ?? '',
      key: editing?.key ?? '',
      type: editing?.type ?? 'text',
      options: joinLines(editing?.options ?? []),
      required: editing?.required ?? false,
    },
  });
  const type = useWatch({ control, name: 'type' });

  async function save(values: Values) {
    setFormError(null);
    const options = values.type === 'list' ? { options: parseLines(values.options) } : {};
    try {
      const saved = editing
        ? unwrap(
            await api.PUT('/api/v1/product-attributes/{attributeId}', {
              params: { path: { attributeId: editing.id } },
              body: { label: values.label, ...options, required: values.required },
            }),
          )
        : unwrap(
            await api.POST('/api/v1/product-attributes', {
              body: { key: values.key, label: values.label, type: values.type, ...options, required: values.required },
            }),
          );
      await queryClient.invalidateQueries({ queryKey: catalogKeys.attributes });
      toast({ tone: 'success', title: 'Atributo guardado', description: saved.label });
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
      open={attribute !== null}
      onOpenChange={(open) => {
        if (!open) {
          setFormError(null);
          onClose();
        }
      }}
      title={editing ? 'Editar atributo' : 'Agregar atributo'}
      description={
        editing
          ? 'La clave y el tipo no cambian, porque los productos guardan sus valores con ellos.'
          : 'Aparece como un campo más en el formulario de producto.'
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
        <Field label="Nombre" hint="Como se ve en el formulario: Principio activo" error={errors.label?.message}>
          <Input autoComplete="off" {...register('label')} />
        </Field>
        <Field
          label="Clave"
          hint={editing ? undefined : 'Minúsculas, sin tildes ni espacios: principio_activo. No se puede cambiar después.'}
          error={errors.key?.message}
        >
          <Input autoComplete="off" spellCheck={false} readOnly={Boolean(editing)} {...register('key')} />
        </Field>
        <Field label="Tipo" error={errors.type?.message}>
          <Controller
            control={control}
            name="type"
            render={({ field }) => (
              <Select
                options={typeOptions}
                value={field.value}
                onValueChange={field.onChange}
                disabled={Boolean(editing)}
              />
            )}
          />
        </Field>
        {type === 'list' && (
          <Field label="Opciones" hint="Una por línea, en el orden en que se ofrecen." error={errors.options?.message}>
            <Textarea rows={5} {...register('options')} />
          </Field>
        )}
        <Checkbox
          label="Obligatorio"
          hint="Todo producto debe tener un valor. Los que no lo tienen lo piden en su próxima edición."
          {...register('required')}
        />
      </form>
    </Dialog>
  );
}
