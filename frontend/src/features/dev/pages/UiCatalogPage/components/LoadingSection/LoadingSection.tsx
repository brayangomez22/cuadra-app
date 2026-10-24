import { Button, EmptyState, IconBox, IconPlus, Inline, Skeleton, Spinner } from '@/shared/ui';
import { Section } from '../Section/Section';
import styles from './LoadingSection.module.scss';

/** What a screen shows while it has nothing yet: EmptyState, Spinner and Skeleton. */
export function LoadingSection() {
  return (
    <>
      <Section title="EmptyState">
        <div className={styles.panel}>
          <EmptyState
            icon={<IconBox />}
            headingLevel={3}
            title="Todavía no hay productos"
            description="Cree el primero o impórtelos desde un archivo de Excel."
            action={<Button icon={<IconPlus />}>Crear producto</Button>}
          />
        </div>
      </Section>

      <Section title="Spinner y Skeleton">
        <Inline gap="3" wrap>
          <Spinner size="sm" />
          <Spinner />
          <Spinner size="lg" label="Buscando productos…" />
        </Inline>
        <div className={styles.skeletons} aria-busy="true">
          <Skeleton />
          <Skeleton />
          <Skeleton className={styles.short} />
          <div className={styles.skeletonBlock}>
            <Skeleton shape="block" />
          </div>
        </div>
      </Section>
    </>
  );
}
