import { useParams } from 'react-router';
import { useCan } from '@/features/auth';
import { documentLabel, useCustomer } from '@/features/customers';
import { QueryState } from '@/shared/query';
import { Page } from '@/shared/ui';
import { useCustomerCredit } from '../../api/credit';
import { AccountActions } from './components/AccountActions/AccountActions';
import { CreditSummary } from './components/CreditSummary/CreditSummary';
import { CreditTerms } from './components/CreditTerms/CreditTerms';
import { OpenReceivables } from './components/OpenReceivables/OpenReceivables';
import { OverdueNotice } from './components/OverdueNotice/OverdueNotice';
import { Statement } from './components/Statement/Statement';
import styles from './CustomerAccountPage.module.scss';

/** A customer's account at /clientes/:id/cuenta: their credit, what they owe, their payments and their statement. */
export function CustomerAccountPage() {
  const { customerId = '' } = useParams();
  const can = useCan();
  const customer = useCustomer(customerId);
  const credit = useCustomerCredit(customerId);
  return (
    <Page
      title={customer.data?.name ?? 'Cuenta del cliente'}
      description={customer.data && documentLabel(customer.data)}
      actions={<AccountActions customerId={customerId} credit={credit.data} />}
    >
      <QueryState queries={[customer, credit]} errorTitle="No pudimos cargar la cuenta del cliente">
        {() =>
          credit.data && (
            <div className={styles.root}>
              <OverdueNotice receivables={credit.data.receivables} />
              <div className={styles.credit}>
                <CreditSummary credit={credit.data} />
                {can('receivables:manage') && <CreditTerms credit={credit.data} />}
              </div>
              <OpenReceivables receivables={credit.data.receivables} />
              <Statement customerId={customerId} />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}
