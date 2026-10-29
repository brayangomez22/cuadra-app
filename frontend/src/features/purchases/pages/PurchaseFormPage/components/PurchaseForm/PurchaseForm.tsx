import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { paths } from '@/app/routes/paths';
import { todayInBogota } from '@/shared/format';
import { Alert, Button, ButtonLink, Page } from '@/shared/ui';
import { purchaseDefaults } from '../../../../model/purchaseForm/purchaseDefaults';
import { purchaseSchema } from '../../../../model/purchaseForm/purchaseSchema';
import type { PurchaseFormData } from '../../hooks/usePurchaseFormData';
import { useSavePurchase } from '../../hooks/useSavePurchase';
import { InvoiceSection } from '../InvoiceSection/InvoiceSection';
import { LinesSection } from '../LinesSection/LinesSection';
import { LiveTotals } from '../LiveTotals/LiveTotals';
import styles from './PurchaseForm.module.scss';

/** The purchase form: the supplier's invoice, its lines and its amounts as they are typed. */
export function PurchaseForm({ data }: { data: PurchaseFormData }) {
  const form = useForm({ resolver: zodResolver(purchaseSchema), defaultValues: purchaseDefaults(todayInBogota()) });
  const { formError, submit } = useSavePurchase(form);
  return (
    <Page
      title="Registrar compra"
      description="La factura de un proveedor: lo que llega entra al inventario de la sede y actualiza el costo."
    >
      <FormProvider {...form}>
        <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
          {formError && <Alert tone="danger">{formError}</Alert>}
          <InvoiceSection data={data} />
          <LinesSection />
          <LiveTotals />
          <div className={styles.footer}>
            <ButtonLink to={paths.purchases.list()} variant="ghost">
              Cancelar
            </ButtonLink>
            <Button type="submit" size="lg" loading={form.formState.isSubmitting}>
              Registrar compra
            </Button>
          </div>
        </form>
      </FormProvider>
    </Page>
  );
}
