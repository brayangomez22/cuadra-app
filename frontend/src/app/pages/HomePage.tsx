import { useAuth } from '@/features/auth';
import { ReorderSummary } from '@/features/inventory';
import { EmptyState, IconHome, Page } from '@/shared/ui';

// The owner's dashboard arrives in T43; for now, a greeting and what to reorder.
export function HomePage() {
  const auth = useAuth();
  const firstName = auth.status === 'authenticated' ? auth.user.name.split(' ')[0] : '';

  return (
    <Page title={firstName ? `Hola, ${firstName}` : 'Inicio'}>
      <ReorderSummary />
      <EmptyState
        icon={<IconHome />}
        title="Aquí verá el resumen del día"
        description="Las ventas, la caja y lo que se está acabando. Mientras tanto, use el menú para ir a cada sección."
      />
    </Page>
  );
}
