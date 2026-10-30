import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { paths } from '@/app/routes/paths';
import { DiscardChangesGuard } from '@/shared/forms/DiscardChangesGuard/DiscardChangesGuard';
import { todayInBogota } from '@/shared/format';
import { Alert, Button, ButtonLink, Page } from '@/shared/ui';
import { purchaseDefaults } from '../../../../model/purchaseForm/purchaseDefaults';
import { purchaseReview } from '../../../../model/purchaseForm/purchaseReview';
import { purchaseSchema } from '../../../../model/purchaseForm/purchaseSchema';
import type { PurchaseFormData } from '../../hooks/usePurchaseFormData';
import { usePurchaseReview } from '../../hooks/usePurchaseReview';
import { useSavePurchase } from '../../hooks/useSavePurchase';
import { ConfirmPurchaseDialog } from '../ConfirmPurchaseDialog/ConfirmPurchaseDialog';
import { InvoiceSection } from '../InvoiceSection/InvoiceSection';
import { LinesSection } from '../LinesSection/LinesSection';
import { LiveTotals } from '../LiveTotals/LiveTotals';
import styles from './PurchaseForm.module.scss';

/**
 * The purchase form: the supplier's invoice, its lines and its amounts as
 * they are typed. It is registered after a review, and leaving it half
 * typed asks first.
 */
export function PurchaseForm({ data }: { data: PurchaseFormData }) {
  const form = useForm({ resolver: zodResolver(purchaseSchema), defaultValues: purchaseDefaults(todayInBogota()) });
  const { formError, save, showError, saved } = useSavePurchase(form);
  const review = usePurchaseReview(form, { save, showError });
  // Read while rendering so react-hook-form keeps it up to date for the guard.
  const { isDirty } = form.formState;
  return (
    <Page
      title="Registrar compra"
      description="La factura de un proveedor: lo que llega entra al inventario de la sede y actualiza el costo."
    >
      <FormProvider {...form}>
        <form className={styles.root} noValidate onSubmit={(event) => void review.open(event)}>
          {formError && <Alert tone="danger">{formError}</Alert>}
          <InvoiceSection data={data} />
          <LinesSection />
          <LiveTotals />
          <div className={styles.footer}>
            <ButtonLink to={paths.purchases.list()} variant="ghost">
              Cancelar
            </ButtonLink>
            <Button type="submit" size="lg">
              Registrar compra
            </Button>
          </div>
        </form>
      </FormProvider>
      <ConfirmPurchaseDialog
        review={review.review && purchaseReview(review.review, data)}
        saving={review.saving}
        onConfirm={() => void review.confirm()}
        onCancel={review.cancel}
        onCloseAutoFocus={review.onCloseAutoFocus}
      />
      <DiscardChangesGuard
        shouldBlock={() => isDirty && !saved.current}
        title="¿Descartar la compra?"
        description="Se pierde lo que lleva escrito de esta factura."
      />
    </Page>
  );
}
