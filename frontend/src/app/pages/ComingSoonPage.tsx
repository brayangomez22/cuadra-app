import { EmptyState, IconBox, Page } from '@/shared/ui';

// ComingSoonPage holds the place of a menu item whose screen is not built yet
// (catalog and inventory in T16, the POS in phase 2).
export function ComingSoonPage({ title }: { title: string }) {
  return (
    <Page title={title}>
      <EmptyState
        icon={<IconBox />}
        title="Disponible pronto"
        description="Esta sección está en construcción. Le avisaremos cuando pueda usarla."
      />
    </Page>
  );
}
