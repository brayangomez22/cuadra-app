import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { z } from 'zod';
import { useLocations } from '@/features/inventory';
import { describeError } from '@/shared/api/errors';
import { formatDateTime, formatMoney } from '@/shared/format';
import {
  Alert,
  Button,
  ButtonLink,
  EmptyState,
  Field,
  IconBox,
  Input,
  Page,
  Select,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from '@/shared/ui';
import { useCurrentCashSession, useOpenCashSession, type CashSession } from './api';
import { moneyField, movementLabels, type CashMovementKind } from './cash';
import { CashMovementDialog } from './CashMovementDialog';
import styles from './CashPage.module.scss';

const title = 'Caja';

/**
 * /caja: the cashier's drawer. Without one open, the form to open it with
 * the float; with one open, its float and cash ins and outs, and the way to
 * close it. The expected cash is never shown here: it comes after the count.
 */
export function CashPage() {
  const current = useCurrentCashSession();
  const locations = useLocations();

  if (current.isPending || locations.isPending) {
    return (
      <Page title={title}>
        <Spinner label="Cargando la caja…" className={styles.loading} />
      </Page>
    );
  }
  if (current.isError || locations.isError) {
    return (
      <Page title={title}>
        <Alert tone="danger" title="No pudimos cargar la caja">
          {describeError(current.error ?? locations.error).message}
        </Alert>
      </Page>
    );
  }

  const session = current.data.session;
  if (session) {
    const location = locations.data.find((l) => l.id === session.location_id);
    return <OpenSession session={session} locationName={location?.name ?? ''} />;
  }

  const active = locations.data.filter((l) => l.active);
  if (active.length === 0) {
    return (
      <Page title={title}>
        <EmptyState
          icon={<IconBox />}
          title="No hay sedes activas"
          description="La caja se abre en una sede. Pida al dueño que cree o active una en Inventario."
        />
      </Page>
    );
  }
  return (
    <Page title={title} description="Cuente el efectivo con el que empieza y abra la caja para vender.">
      <OpenForm locations={active.map((l) => ({ value: l.id, label: l.name }))} />
    </Page>
  );
}

const openSchema = z.object({
  location_id: z.string().min(1, 'Elija la sede de la caja'),
  opening_float: moneyField('nonnegative', 'Escriba la base, aunque sea 0'),
});
type OpenInput = z.input<typeof openSchema>;
type OpenValues = z.output<typeof openSchema>;

function OpenForm({ locations }: { locations: { value: string; label: string }[] }) {
  const navigate = useNavigate();
  const open = useOpenCashSession();
  const [formError, setFormError] = useState<string>();
  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<OpenInput, unknown, OpenValues>({
    resolver: zodResolver(openSchema),
    defaultValues: {
      location_id: locations.length === 1 ? (locations[0]?.value ?? '') : '',
      opening_float: '',
    },
  });

  function submit(values: OpenValues) {
    setFormError(undefined);
    open.mutate(values, {
      // Opening the drawer is the step before selling.
      onSuccess: () => navigate('/vender'),
      onError: (error) => setFormError(describeError(error).message),
    });
  }

  return (
    <form className={styles.form} noValidate onSubmit={handleSubmit(submit)}>
      {formError && <Alert tone="danger">{formError}</Alert>}
      {locations.length > 1 && (
        <Field label="Sede" error={errors.location_id?.message}>
          <Controller
            control={control}
            name="location_id"
            render={({ field }) => (
              <Select
                options={locations}
                value={field.value || undefined}
                onValueChange={field.onChange}
                placeholder="Elija la sede"
              />
            )}
          />
        </Field>
      )}
      {locations.length === 1 && errors.location_id && <Alert tone="danger">{errors.location_id.message}</Alert>}
      <Field
        label="Base inicial"
        hint="El efectivo que hay en el cajón al empezar."
        error={errors.opening_float?.message}
      >
        <Input inputMode="decimal" align="end" autoComplete="off" autoFocus {...register('opening_float')} />
      </Field>
      <div className={styles.footer}>
        <Button type="submit" size="lg" loading={open.isPending}>
          Abrir caja
        </Button>
      </div>
    </form>
  );
}

function OpenSession({ session, locationName }: { session: CashSession; locationName: string }) {
  const [movement, setMovement] = useState<CashMovementKind>();

  return (
    <Page
      title={title}
      description="Registre aquí el efectivo que entra o sale del cajón sin ser una venta."
      actions={
        <ButtonLink to="/caja/cerrar" size="lg">
          Cerrar caja
        </ButtonLink>
      }
    >
      <dl className={styles.facts}>
        <div className={styles.fact}>
          <dt>Sede</dt>
          <dd>{locationName}</dd>
        </div>
        <div className={styles.fact}>
          <dt>Abierta</dt>
          <dd>{formatDateTime(session.opened_at)}</dd>
        </div>
        <div className={styles.fact}>
          <dt>Base inicial</dt>
          <dd className={styles.amount}>{formatMoney(session.opening_float)}</dd>
        </div>
      </dl>

      <div className={styles.movementActions}>
        <Button variant="secondary" onClick={() => setMovement('in')}>
          Registrar ingreso
        </Button>
        <Button variant="secondary" onClick={() => setMovement('out')}>
          Registrar egreso
        </Button>
      </div>

      {session.movements.length === 0 ? (
        <p className={styles.note}>Todavía no hay ingresos ni egresos.</p>
      ) : (
        <Table caption="Ingresos y egresos">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Hora</TableHeaderCell>
              <TableHeaderCell>Tipo</TableHeaderCell>
              <TableHeaderCell>Motivo</TableHeaderCell>
              <TableHeaderCell numeric>Valor</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {session.movements.map((m) => (
              <TableRow key={m.id}>
                <TableCell>{formatDateTime(m.created_at)}</TableCell>
                <TableCell>{movementLabels[m.kind]}</TableCell>
                <TableCell>{m.reason}</TableCell>
                <TableCell numeric>{formatMoney(m.kind === 'out' ? `-${m.amount}` : m.amount)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CashMovementDialog
        sessionId={session.id}
        kind={movement}
        onOpenChange={(open) => {
          if (!open) setMovement(undefined);
        }}
      />
    </Page>
  );
}
