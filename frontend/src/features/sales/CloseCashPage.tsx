import { useQueryClient } from '@tanstack/react-query';
import { useId, useState, type FormEvent } from 'react';
import { Navigate } from 'react-router';
import { describeError } from '@/shared/api/errors';
import { formatMoney } from '@/shared/format';
import { Alert, Button, Dialog, DialogClose, Field, Input, Page, Spinner, Textarea } from '@/shared/ui';
import { salesKeys, useCloseCashSession, useCurrentCashSession, type CashSession } from './api';
import { countedTotal, denominationLabel, denominations, parseCount, type DenominationCount } from './cash';
import styles from './CloseCashPage.module.scss';
import { paths } from '@/app/routes/paths';

const title = 'Cerrar caja';
const description =
  'Cuente el efectivo del cajón por billetes y monedas. Al cerrar, Cuadra lo compara con lo que debería haber.';

/**
 * /caja/cerrar: the count of the drawer (arqueo). It is blind: the screen
 * adds up what was counted, and the expected cash only shows in the report,
 * after closing.
 */
export function CloseCashPage() {
  const queryClient = useQueryClient();
  const current = useCurrentCashSession();
  const [closedId, setClosedId] = useState<string>();

  // Once closed, the user has no open session: go to the report, not to /caja.
  if (closedId) return <Navigate to={paths.sales.cashReport(closedId)} replace />;
  if (current.isPending) {
    return (
      <Page title={title}>
        <Spinner label="Cargando la caja…" className={styles.loading} />
      </Page>
    );
  }
  if (current.isError) {
    return (
      <Page title={title}>
        <Alert tone="danger" title="No pudimos cargar la caja">
          {describeError(current.error).message}
        </Alert>
      </Page>
    );
  }
  const session = current.data.session;
  if (!session) return <Navigate to={paths.sales.cash()} replace />;
  return (
    <Page title={title} description={description}>
      <CountForm
        session={session}
        onClosed={(id) => {
          setClosedId(id);
          void queryClient.invalidateQueries({ queryKey: salesKeys.currentCash });
        }}
      />
    </Page>
  );
}

type Read = { counts: DenominationCount[]; invalid: Set<string> };

function read(texts: Record<string, string>): Read {
  const counts: DenominationCount[] = [];
  const invalid = new Set<string>();
  for (const d of denominations) {
    const parsed = parseCount(texts[d.value] ?? '');
    if (!parsed.ok) invalid.add(d.value);
    else if (parsed.quantity > 0) counts.push({ denomination: d.value, quantity: parsed.quantity });
  }
  return { counts, invalid };
}

function CountForm({ session, onClosed }: { session: CashSession; onClosed: (id: string) => void }) {
  const close = useCloseCashSession(session.id);
  const [texts, setTexts] = useState<Record<string, string>>({});
  const [notes, setNotes] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const totalId = useId();

  const { counts, invalid } = read(texts);
  const total = countedTotal(counts);

  function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitted(true);
    if (invalid.size > 0) return;
    close.reset();
    setConfirming(true);
  }

  function confirm() {
    const trimmed = notes.trim();
    close.mutate({ counts, ...(trimmed && { notes: trimmed }) }, { onSuccess: (closed) => onClosed(closed.id) });
  }

  return (
    <form className={styles.root} noValidate onSubmit={submit}>
      <div className={styles.count}>
        <fieldset className={styles.counts}>
          <legend className="srOnly">Conteo por denominación</legend>
          {denominations.map((d) => {
            const parsed = parseCount(texts[d.value] ?? '');
            const subtotal = parsed.ok ? (BigInt(d.value) * BigInt(parsed.quantity)).toString() : undefined;
            return (
              <div key={d.value} className={styles.denomination}>
                <Field
                  label={denominationLabel(d)}
                  error={submitted && invalid.has(d.value) ? 'Escriba cuántos hay, sin decimales' : undefined}
                >
                  <Input
                    inputMode="numeric"
                    align="end"
                    autoComplete="off"
                    value={texts[d.value] ?? ''}
                    onChange={(event) => setTexts((t) => ({ ...t, [d.value]: event.target.value }))}
                  />
                </Field>
                <span className={styles.subtotal} aria-hidden="true">
                  {subtotal && subtotal !== '0' ? formatMoney(subtotal) : ''}
                </span>
              </div>
            );
          })}
        </fieldset>
        <Field label="Notas" optional hint="Lo que el dueño deba saber de este cierre.">
          <Textarea maxLength={500} value={notes} onChange={(event) => setNotes(event.target.value)} />
        </Field>
      </div>

      <div className={styles.summary}>
        <p className={styles.total} role="status" aria-labelledby={totalId}>
          <span id={totalId} className={styles.totalLabel}>
            Total contado
          </span>
          <span className={styles.totalValue}>{formatMoney(total)}</span>
        </p>
        <Button type="submit" size="lg" className={styles.submit}>
          Cerrar caja
        </Button>
      </div>

      <Dialog
        open={confirming}
        onOpenChange={setConfirming}
        title="¿Cerrar la caja?"
        description={`Contó ${formatMoney(total)}. Después de cerrarla no se puede modificar.`}
        footer={
          <>
            {/* The safe choice has the focus: closing cannot be undone. */}
            <DialogClose asChild>
              <Button variant="secondary" autoFocus>
                Seguir contando
              </Button>
            </DialogClose>
            <Button onClick={confirm} loading={close.isPending}>
              Cerrar caja
            </Button>
          </>
        }
      >
        {close.isError && <Alert tone="danger">{describeError(close.error).message}</Alert>}
      </Dialog>
    </form>
  );
}
