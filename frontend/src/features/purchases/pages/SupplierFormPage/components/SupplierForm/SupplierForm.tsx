import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { paths } from '@/app/routes/paths';
import { formatMoney } from '@/shared/format';
import { Alert, Button, ButtonLink, Page } from '@/shared/ui';
import type { Supplier } from '../../../../api/types';
import { supplierDefaults } from '../../../../model/supplierForm/supplierDefaults';
import { supplierSchema } from '../../../../model/supplierForm/supplierSchema';
import { useSaveSupplierForm } from '../../hooks/useSaveSupplierForm';
import { ContactSection } from '../ContactSection/ContactSection';
import { IdentitySection } from '../IdentitySection/IdentitySection';
import { TermsSection } from '../TermsSection/TermsSection';
import styles from './SupplierForm.module.scss';

/** The supplier form: who it is, how to reach it and the credit it gives. */
export function SupplierForm({ supplier }: { supplier: Supplier | undefined }) {
  const form = useForm({ resolver: zodResolver(supplierSchema), defaultValues: supplierDefaults(supplier) });
  const { formError, submit } = useSaveSupplierForm(form, supplier);
  return (
    <Page
      title={supplier ? 'Editar proveedor' : 'Nuevo proveedor'}
      description={supplier && `${supplier.name} · Se le debe ${formatMoney(supplier.balance_due)}`}
    >
      <FormProvider {...form}>
        <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
          {formError && <Alert tone="danger">{formError}</Alert>}
          <IdentitySection />
          <ContactSection />
          <TermsSection editing={Boolean(supplier)} />
          <div className={styles.footer}>
            <ButtonLink to={paths.purchases.suppliers()} variant="ghost">
              Cancelar
            </ButtonLink>
            <Button type="submit" size="lg" loading={form.formState.isSubmitting}>
              Guardar proveedor
            </Button>
          </div>
        </form>
      </FormProvider>
    </Page>
  );
}
