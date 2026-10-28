import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { paths } from '@/app/routes/paths';
import { Alert, Button, ButtonLink, FormGrid, FormSection, Page } from '@/shared/ui';
import { IdentityFields } from '../../../../components/IdentityFields/IdentityFields';
import { customerDefaults } from '../../../../model/customerForm/customerDefaults';
import { customerSchema } from '../../../../model/customerForm/customerSchema';
import type { CustomerFormData } from '../../hooks/useCustomerFormData';
import { useSaveCustomerForm } from '../../hooks/useSaveCustomerForm';
import { ContactSection } from '../ContactSection/ContactSection';
import { TaxSection } from '../TaxSection/TaxSection';
import { TradeNameField } from './TradeNameField';
import styles from './CustomerForm.module.scss';

/** The customer form: who it is, how to reach it and how it is classified. */
export function CustomerForm({ data: { customer, segments } }: { data: CustomerFormData }) {
  const form = useForm({ resolver: zodResolver(customerSchema), defaultValues: customerDefaults(customer) });
  const { formError, submit } = useSaveCustomerForm(form, customer);
  return (
    <Page title={customer ? 'Editar cliente' : 'Nuevo cliente'} description={customer?.name}>
      <FormProvider {...form}>
        <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
          {formError && <Alert tone="danger">{formError}</Alert>}
          <FormSection title="Identificación" hint="Los datos del RUT o del documento, como deben salir en la factura.">
            <FormGrid>
              <IdentityFields />
              <TradeNameField />
            </FormGrid>
          </FormSection>
          <ContactSection />
          <TaxSection segments={segments} editing={Boolean(customer)} />
          <div className={styles.footer}>
            <ButtonLink to={paths.customers.list()} variant="ghost">
              Cancelar
            </ButtonLink>
            <Button type="submit" size="lg" loading={form.formState.isSubmitting}>
              Guardar cliente
            </Button>
          </div>
        </form>
      </FormProvider>
    </Page>
  );
}
