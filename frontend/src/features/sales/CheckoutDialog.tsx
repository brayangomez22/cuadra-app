import { useState, type FormEvent } from 'react';
import { formatMoney, isNegative, isZero, parseMoneyInput, toDecimalInput } from '@/shared/format';
import { Alert, Button, Dialog, DialogClose, Field, IconButton, IconClose, IconPlus, IconPrinter, Input, Select } from '@/shared/ui';
import { methodLabels, settle, type PaymentInput, type PaymentMethod, type Settlement } from './cart';
import type { Sale } from './api';
import type { Paper } from './Receipt';
import styles from './CheckoutDialog.module.scss';

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The total of the cart, IVA included. */
  total: string;
  /** Sends the sale; the payments are already checked against the total. */
  onConfirm: (payments: PaymentInput[]) => void;
  pending: boolean;
  /** Why the API rejected the sale, in Spanish. */
  error?: string;
  /** The sale the API confirmed: the dialog shows its number and change. */
  sale?: Sale;
  /** Closes the result and starts the next sale. */
  onNewSale: () => void;
  /** The receipt's paper width and how to change it. */
  paper: Paper;
  onPaperChange: (paper: Paper) => void;
  /** Prints the confirmed sale's receipt. */
  onPrint: () => void;
};

const paperOptions = [
  { value: '80', label: '80 mm' },
  { value: '58', label: '58 mm' },
];

const methodOptions = Object.entries(methodLabels).map(([value, label]) => ({ value, label }));
// The order in which a new payment proposes a method it does not have yet.
const nextMethods: PaymentMethod[] = ['card', 'nequi', 'daviplata', 'transfer', 'cash'];
const MAX_PAYMENTS = 10;

type Row = { key: number; method: PaymentMethod; text: string };
type ReadRow = { amount?: string; error?: string };

/**
 * Reads the amounts typed. A single empty payment is the exact total, so the
 * cashier charges an exact amount with Enter alone.
 */
function readRows(rows: Row[], total: string): ReadRow[] {
  return rows.map((row) => {
    if (row.text.trim() === '') {
      return rows.length === 1 ? { amount: total } : { error: 'Escriba el valor' };
    }
    const parsed = parseMoneyInput(row.text);
    if (!parsed.ok) return { error: 'Escriba un valor, por ejemplo 50.000 o 50000' };
    if (isZero(parsed.value) || isNegative(parsed.value)) return { error: 'El valor debe ser mayor que cero' };
    return { amount: parsed.value };
  });
}

function settlementError(s: Settlement): string | undefined {
  if (s.error === 'insufficient') return `Faltan ${formatMoney(s.missing)}.`;
  if (s.error === 'non_cash_overpayment') {
    return 'Solo el efectivo da cambio: tarjeta, Nequi, Daviplata y transferencia no pueden sumar más que el total.';
  }
  return undefined;
}

/** Takes the payments of the sale, mixed if needed, and shows the change. */
export function CheckoutDialog({
  open,
  onOpenChange,
  total,
  onConfirm,
  pending,
  error,
  sale,
  onNewSale,
  paper,
  onPaperChange,
  onPrint,
}: Props) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      size="md"
      title={sale ? `Venta N.º ${sale.number}` : 'Cobrar'}
      description={sale ? 'La venta quedó registrada.' : 'Con un solo pago en efectivo vacío se cobra el valor exacto.'}
    >
      {/* Mounted per opening: every charge starts with one empty cash payment. */}
      {open && (sale ? (
        <SaleDone sale={sale} onNewSale={onNewSale} paper={paper} onPaperChange={onPaperChange} onPrint={onPrint} />
      ) : (
        <PaymentsForm total={total} onConfirm={onConfirm} pending={pending} error={error} />
      ))}
    </Dialog>
  );
}

function PaymentsForm({ total, onConfirm, pending, error }: Pick<Props, 'total' | 'onConfirm' | 'pending' | 'error'>) {
  const [rows, setRows] = useState<Row[]>([{ key: 0, method: 'cash', text: '' }]);
  const [submitted, setSubmitted] = useState(false);
  const [problem, setProblem] = useState<string>();

  const read = readRows(rows, total);
  const valid = read.flatMap((r, i) => {
    const method = rows[i]?.method;
    return r.amount && method ? [{ method, amount: r.amount }] : [];
  });
  const settlement = settle(total, valid);

  function change(key: number, changes: Partial<Row>) {
    setRows((current) => current.map((r) => (r.key === key ? { ...r, ...changes } : r)));
    setProblem(undefined);
  }

  function addPayment() {
    const used = new Set(rows.map((r) => r.method));
    const method = nextMethods.find((m) => !used.has(m)) ?? 'cash';
    const text = isZero(settlement.missing) ? '' : toDecimalInput(settlement.missing.replace(/\.00$/, ''));
    setRows((current) => [...current, { key: Math.max(...current.map((r) => r.key)) + 1, method, text }]);
    setProblem(undefined);
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitted(true);
    if (pending || read.some((r) => r.error)) return;
    const message = settlementError(settlement);
    setProblem(message);
    if (!message) onConfirm(valid);
  }

  return (
    <form className={styles.root} onSubmit={submit} noValidate>
      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt>Total</dt>
          <dd className={styles.total}>{formatMoney(total)}</dd>
        </div>
        <div className={styles.figure}>
          <dt>Recibido</dt>
          <dd>{formatMoney(settlement.paid)}</dd>
        </div>
        {isZero(settlement.missing) ? (
          <div className={styles.figure} data-tone={isZero(settlement.change) ? undefined : 'change'}>
            <dt>Cambio</dt>
            <dd className={styles.change}>{formatMoney(settlement.change)}</dd>
          </div>
        ) : (
          <div className={styles.figure} data-tone="missing">
            <dt>Falta</dt>
            <dd className={styles.change}>{formatMoney(settlement.missing)}</dd>
          </div>
        )}
      </dl>

      <fieldset className={styles.payments}>
        <legend className="srOnly">Pagos</legend>
        {rows.map((row, index) => {
          const n = index + 1;
          return (
            <div key={row.key} className={styles.payment}>
              <Field className={styles.method} label={<>Medio de pago <span className="srOnly">{n}</span></>}>
                <Select
                  options={methodOptions}
                  value={row.method}
                  onValueChange={(value) => change(row.key, { method: value as PaymentMethod })}
                />
              </Field>
              <Field
                className={styles.amount}
                label={<>Valor <span className="srOnly">del pago {n}</span></>}
                error={submitted ? read[index]?.error : undefined}
              >
                <Input
                  align="end"
                  inputMode="decimal"
                  autoComplete="off"
                  placeholder={rows.length === 1 ? toDecimalInput(total.replace(/\.00$/, '')) : undefined}
                  value={row.text}
                  onChange={(event) => change(row.key, { text: event.target.value })}
                  autoFocus={index === 0}
                />
              </Field>
              {index > 0 && (
                <IconButton
                  className={styles.remove}
                  label={`Quitar pago ${n}`}
                  icon={<IconClose />}
                  onClick={() => setRows((current) => current.filter((r) => r.key !== row.key))}
                />
              )}
            </div>
          );
        })}
        {rows.length < MAX_PAYMENTS && (
          <Button variant="ghost" icon={<IconPlus />} className={styles.add} onClick={addPayment}>
            Agregar medio de pago
          </Button>
        )}
      </fieldset>

      {problem && <Alert tone="danger">{problem}</Alert>}
      {error && !problem && <Alert tone="danger">{error}</Alert>}

      <div className={styles.footer}>
        <DialogClose asChild>
          <Button variant="secondary">Volver</Button>
        </DialogClose>
        <Button type="submit" size="lg" loading={pending}>
          Confirmar venta
        </Button>
      </div>
    </form>
  );
}

function SaleDone({
  sale,
  onNewSale,
  paper,
  onPaperChange,
  onPrint,
}: { sale: Sale } & Pick<Props, 'onNewSale' | 'paper' | 'onPaperChange' | 'onPrint'>) {
  return (
    <div className={styles.root}>
      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt>Total</dt>
          <dd>{formatMoney(sale.totals.total)}</dd>
        </div>
        <div className={styles.figure}>
          <dt>Recibido</dt>
          <dd>{formatMoney(sale.paid)}</dd>
        </div>
        <div className={styles.figure} data-tone="change">
          <dt>Cambio</dt>
          <dd className={styles.total}>{formatMoney(sale.change)}</dd>
        </div>
      </dl>
      <div className={styles.footer}>
        <div className={styles.print}>
          <Field label="Papel" className={styles.paper}>
            <Select
              options={paperOptions}
              value={paper}
              onValueChange={(value) => onPaperChange(value === '58' ? '58' : '80')}
            />
          </Field>
          <Button variant="secondary" size="lg" icon={<IconPrinter />} onClick={onPrint}>
            Imprimir tirilla
          </Button>
        </div>
        {/* The next sale keeps the focus: Enter starts it, as before printing existed. */}
        <Button size="lg" onClick={onNewSale} autoFocus>
          Nueva venta
        </Button>
      </div>
    </div>
  );
}
