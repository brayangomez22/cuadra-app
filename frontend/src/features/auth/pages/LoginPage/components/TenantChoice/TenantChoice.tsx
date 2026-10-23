import { Alert, Button, Stack } from '@/shared/ui';
import type { TenantOption } from '../../../../api/types';
import { AuthPage } from '../../../../components/AuthPage/AuthPage';
import styles from './TenantChoice.module.scss';

type Props = {
  tenants: TenantOption[];
  formError: string | null;
  /** The company being entered, while its login runs. */
  pendingTenant: string | null;
  onChoose: (tenantId: string) => void;
  onBack: () => void;
};

export function TenantChoice({ tenants, formError, pendingTenant, onChoose, onBack }: Props) {
  return (
    <AuthPage
      title="¿A qué empresa quiere entrar?"
      description="Su correo está registrado en varias empresas. Elija una para continuar."
    >
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Stack gap="3">
        {tenants.map((tenant) => (
          <Button
            key={tenant.id}
            variant="secondary"
            size="lg"
            className={styles.tenant}
            loading={pendingTenant === tenant.id}
            disabled={pendingTenant !== null}
            onClick={() => {
              onChoose(tenant.id);
            }}
          >
            {tenant.name}
          </Button>
        ))}
      </Stack>
      <Button variant="ghost" className={styles.back} disabled={pendingTenant !== null} onClick={onBack}>
        Usar otro correo
      </Button>
    </AuthPage>
  );
}
