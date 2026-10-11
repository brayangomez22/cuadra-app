import { useAuth } from '@/features/auth';
import { EmptyState, IconHome, Page } from '@/shared/ui';

// The owner's dashboard arrives in T43; for now, a greeting.
export function HomePage() {
  const auth = useAuth();
  const firstName = auth.status === 'authenticated' ? auth.user.name.split(' ')[0] : '';

  return (
    <Page title={firstName ? `Hola, ${firstName}` : 'Inicio'}>
      <EmptyState
        icon={<IconHome />}
        title="Aquí verá el resumen del día"
        description="Las ventas, la caja y lo que se está acabando. Mientras tanto, use el menú para ir a cada sección."
      />
    </Page>
  );
}
