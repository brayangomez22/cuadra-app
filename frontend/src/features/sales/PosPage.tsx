import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo, useReducer, useRef, useState } from 'react';
import { useAuth } from '@/features/auth';
import { inventoryKeys, useLocations } from '@/features/inventory';
import { ApiError, describeError } from '@/shared/api/errors';
import { formatMoney, formatPercent, isZero } from '@/shared/format';
import {
  Alert,
  Button,
  ButtonLink,
  Dialog,
  DialogClose,
  EmptyState,
  Field,
  IconBox,
  IconCash,
  Page,
  Select,
  Spinner,
  TextLink,
} from '@/shared/ui';
import { salesKeys, useConfirmSale, useCurrentCashSession, useDiscountLimit, type Sale } from './api';
import { CartLines } from './CartLines';
import { cartReducer, cartTotals, type DiscountLimit, type PaymentInput } from './cart';
import { CheckoutDialog } from './CheckoutDialog';
import { PrintableReceipt } from './PrintableReceipt';
import type { Paper } from './Receipt';
import { ProductSearch } from './ProductSearch';
import { loadCart, loadLocation, loadPaper, saveCart, saveLocation, savePaper } from './storage';
import styles from './PosPage.module.scss';

// Until the API answers, nobody discounts: the safe default (docs/decisiones.md).
const noDiscount: DiscountLimit = { unlimited: false, max: '0' };

// Errors of a sale that mean the cashier's drawer is gone: the screen asks
// for it again once the checkout closes.
const cashSessionErrors = new Set(['cash_session_required', 'cash_session_closed']);

/**
 * The point of sale: search or scan, adjust the cart and charge, all from the
 * keyboard. F2 goes to the search, F4 charges and Escape cancels the sale.
 * Sales go to the cashier's open cash session, at its location; if the
 * tenant requires one and there is none, the screen asks to open it.
 */
export function PosPage() {
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const cartKey = `cuadra.pos.cart.${user?.id ?? ''}`;
  const locationKey = `cuadra.pos.location.${user?.tenant_id ?? ''}`;

  const queryClient = useQueryClient();
  const locations = useLocations();
  const limit = useDiscountLimit();
  const cash = useCurrentCashSession();
  const confirmSale = useConfirmSale();

  const [cart, dispatch] = useReducer(cartReducer, cartKey, loadCart);
  const [storedLocation, setStoredLocation] = useState(() => loadLocation(locationKey));
  const [checkoutOpen, setCheckoutOpen] = useState(false);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [sale, setSale] = useState<Sale>();
  const [paper, setPaper] = useState<Paper>(loadPaper);
  const [outOfStock, setOutOfStock] = useState<string>();
  const [cashLost, setCashLost] = useState(false);
  const search = useRef<HTMLInputElement>(null);

  const activeLocations = useMemo(() => (locations.data ?? []).filter((l) => l.active), [locations.data]);
  const [onlyLocation] = activeLocations;
  const session = cash.data?.session;
  // With an open drawer, the sale happens where the drawer is.
  const locationId = session
    ? session.location_id
    : activeLocations.some((l) => l.id === storedLocation)
      ? storedLocation
      : activeLocations.length === 1
        ? onlyLocation?.id
        : undefined;

  const discountLimit = limit.data ?? noDiscount;
  const showDiscount = discountLimit.unlimited || !isZero(discountLimit.max);
  const totals = cartTotals(cart, discountLimit);
  const canCharge = totals.valid && Boolean(locationId);
  // A line with an invalid field is left out of the total until it is fixed.
  const invalidLines = totals.lines.filter((l) => !l.input).length;

  useEffect(() => saveCart(cartKey, cart), [cartKey, cart]);

  const focusSearch = () => requestAnimationFrame(() => search.current?.focus());

  // The screen's shortcuts. Open dialogs handle their own keys (Escape closes them).
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented || checkoutOpen || cancelOpen) return;
      if (event.key === 'F2') {
        event.preventDefault();
        search.current?.focus();
      } else if (event.key === 'F4') {
        event.preventDefault();
        if (canCharge) openCheckout();
      } else if (event.key === 'Escape' && cart.lines.length > 0) {
        event.preventDefault();
        setCancelOpen(true);
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  });

  function change(action: Parameters<typeof dispatch>[0]) {
    dispatch(action);
    setOutOfStock(undefined);
  }

  function openCheckout() {
    confirmSale.reset();
    setSale(undefined);
    setCheckoutOpen(true);
  }

  function closeCheckout() {
    setCheckoutOpen(false);
    if (cashLost) {
      setCashLost(false);
      void queryClient.invalidateQueries({ queryKey: salesKeys.currentCash });
    }
    if (sale) {
      setSale(undefined);
      confirmSale.reset();
    }
    focusSearch();
  }

  function confirm(payments: PaymentInput[]) {
    if (!locationId) return;
    const lines = totals.lines.flatMap((l) => (l.input ? [l.input] : []));
    confirmSale.mutate(
      { location_id: locationId, lines, payments },
      {
        onSuccess: (confirmed) => {
          setSale(confirmed);
          change({ type: 'clear' });
          // The sale took stock: the inventory screens read it again.
          void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
        },
        onError: (error) => {
          if (error instanceof ApiError && error.code === 'insufficient_stock') {
            setOutOfStock(error.details.product_id);
          }
          if (error instanceof ApiError && cashSessionErrors.has(error.code ?? '')) setCashLost(true);
        },
      },
    );
  }

  if (locations.isPending || cash.isPending) {
    return (
      <Page title="Vender">
        <Spinner label="Cargando la caja…" />
      </Page>
    );
  }
  if (locations.isError || cash.isError) {
    return (
      <Page title="Vender">
        <Alert tone="danger" title="No pudimos abrir la caja">
          {describeError(locations.error ?? cash.error).message}
        </Alert>
      </Page>
    );
  }
  if (cash.data.required && !session) {
    return (
      <Page title="Vender">
        <EmptyState
          icon={<IconCash />}
          title="Abra la caja para vender"
          description="Cuente el efectivo del cajón y abra la caja. La venta en curso no se pierde."
          action={<ButtonLink to="/caja">Abrir caja</ButtonLink>}
        />
      </Page>
    );
  }
  if (activeLocations.length === 0) {
    return (
      <Page title="Vender">
        <EmptyState
          icon={<IconBox />}
          title="No hay sedes activas para vender"
          description="Las ventas descuentan el stock de una sede. Pida al dueño que cree o active una en Inventario."
        />
      </Page>
    );
  }

  return (
    <Page
      title="Vender"
      className={styles.root}
      actions={
        session ? (
          <p className={styles.cash}>
            Caja abierta en {locations.data.find((l) => l.id === session.location_id)?.name}
            {' · '}
            <TextLink to="/caja">Ver caja</TextLink>
          </p>
        ) : (
          activeLocations.length > 1 && (
            <Field label="Sede" className={styles.location}>
              <Select
                options={activeLocations.map((l) => ({ value: l.id, label: l.name }))}
                value={locationId}
                placeholder="Elija la sede"
                onValueChange={(value) => {
                  setStoredLocation(value);
                  saveLocation(locationKey, value);
                  focusSearch();
                }}
              />
            </Field>
          )
        )
      }
    >
      <div className={styles.workspace}>
        <div className={styles.sale}>
          <ProductSearch ref={search} onAdd={(product, quantity) => change({ type: 'add', product, quantity })} />
          {showDiscount && !discountLimit.unlimited && (
            <p className={styles.note}>Descuento máximo por línea: {formatPercent(discountLimit.max, 'rate')}</p>
          )}
          <CartLines
            totals={totals}
            dispatch={change}
            showDiscount={showDiscount}
            outOfStock={outOfStock}
            onDone={focusSearch}
          />
        </div>

        <section className={styles.summary} aria-label="Resumen de la venta">
          <dl className={styles.figures}>
            <div className={styles.figure} data-detail="true">
              <dt>Subtotal</dt>
              <dd>{formatMoney(totals.subtotal)}</dd>
            </div>
            {!isZero(totals.discounts) && (
              <div className={styles.figure} data-detail="true">
                <dt>Descuentos</dt>
                <dd>{formatMoney(totals.discounts)}</dd>
              </div>
            )}
            <div className={styles.figure} data-detail="true">
              <dt>IVA</dt>
              <dd>{formatMoney(totals.tax)}</dd>
            </div>
            <div className={styles.figure} data-role="total">
              <dt>Total</dt>
              <dd className={styles.total}>{formatMoney(totals.total)}</dd>
            </div>
          </dl>
          {!locationId && <p className={styles.note}>Elija la sede para cobrar.</p>}
          {invalidLines > 0 && (
            <p className={styles.note} role="status">
              Corrija {invalidLines === 1 ? '1 línea' : `${invalidLines} líneas`} para cobrar.
            </p>
          )}
          <Button size="lg" className={styles.charge} disabled={!canCharge} onClick={openCheckout} aria-keyshortcuts="F4">
            Cobrar <kbd className={styles.key} aria-hidden="true">F4</kbd>
          </Button>
          <p className={styles.shortcuts} aria-hidden="true">
            <kbd>F2</kbd> buscar · <kbd>F4</kbd> cobrar · <kbd>Esc</kbd> cancelar
          </p>
        </section>
      </div>

      <CheckoutDialog
        open={checkoutOpen}
        onOpenChange={(open) => (open ? setCheckoutOpen(true) : closeCheckout())}
        total={totals.total}
        onConfirm={confirm}
        pending={confirmSale.isPending}
        error={confirmSale.isError ? describeError(confirmSale.error).message : undefined}
        sale={sale}
        onNewSale={closeCheckout}
        paper={paper}
        onPaperChange={(value) => {
          setPaper(value);
          savePaper(value);
        }}
        onPrint={() => window.print()}
      />

      {sale && user && (
        <PrintableReceipt
          sale={sale}
          business={{ name: user.tenant_name, nit: user.tenant_nit }}
          location={locations.data?.find((l) => l.id === sale.location_id)?.name ?? ''}
          cashier={user.name}
          paper={paper}
        />
      )}

      <Dialog
        open={cancelOpen}
        onOpenChange={(open) => {
          setCancelOpen(open);
          if (!open) focusSearch();
        }}
        title="¿Cancelar la venta?"
        description="Se quitan todos los productos del carrito. No se guarda nada."
        footer={
          <>
            {/* The safe choice has the focus: the Enter of a scan keeps the sale. */}
            <DialogClose asChild>
              <Button variant="secondary" autoFocus>
                Seguir vendiendo
              </Button>
            </DialogClose>
            <Button
              variant="danger"
              onClick={() => {
                change({ type: 'clear' });
                setCancelOpen(false);
                focusSearch();
              }}
            >
              Cancelar venta
            </Button>
          </>
        }
      />
    </Page>
  );
}
