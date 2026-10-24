import { ButtonsSection } from './components/ButtonsSection/ButtonsSection';
import { FormFieldsSection } from './components/FormFieldsSection/FormFieldsSection';
import { LayoutSamples } from './components/LayoutSamples/LayoutSamples';
import { LoadingSection } from './components/LoadingSection/LoadingSection';
import { OverlaysSection } from './components/OverlaysSection/OverlaysSection';
import { Section } from './components/Section/Section';
import { StatusSection } from './components/StatusSection/StatusSection';
import { StockTableSection } from './components/StockTableSection/StockTableSection';
import { ThemeField } from './components/ThemeField/ThemeField';
import { ToastSection } from './components/ToastSection/ToastSection';
import styles from './UiCatalogPage.module.scss';

// Internal catalog of shared/ui in all its states. Only in development
// (routes.tsx): it is the visual reference for reviews and screenshots.
export function UiCatalogPage() {
  return (
    <main className={styles.root}>
      <header className={styles.header}>
        <div>
          <h1 className={styles.title}>Componentes base</h1>
          <p className={styles.lead}>
            Catálogo interno de <code>shared/ui</code>. Solo existe en desarrollo.
          </p>
        </div>
        <ThemeField />
      </header>
      <ButtonsSection />
      <FormFieldsSection />
      <StockTableSection />
      <StatusSection />
      <OverlaysSection />
      <ToastSection />
      <LoadingSection />
      <Section title="Stack e Inline">
        <LayoutSamples />
      </Section>
    </main>
  );
}
