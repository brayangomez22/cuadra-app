import { QueryState } from '@/shared/query';
import { Stack } from '@/shared/ui';
import { useSalesRules, useStockRules } from '../../api/businessRules';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import { BusinessContactForm } from './components/BusinessContactForm/BusinessContactForm';
import { DefaultTaxForm } from './components/DefaultTaxForm/DefaultTaxForm';
import { SalesRulesForm } from './components/SalesRulesForm/SalesRulesForm';
import { StockRulesForm } from './components/StockRulesForm/StockRulesForm';

/** /configuracion/general: the business's own data and rules. Owner only (tenant:manage). */
export function GeneralSettings() {
  const sales = useSalesRules();
  const stock = useStockRules();

  return (
    <SettingsSection title="General" description="Los datos de su negocio y las reglas con las que trabaja su equipo.">
      <Stack gap="8">
        <BusinessContactForm />
        <DefaultTaxForm />
        <QueryState queries={[sales, stock]} errorTitle="No pudimos cargar las reglas del negocio">
          {() => (
            <>
              {sales.data && <SalesRulesForm rules={sales.data} />}
              {stock.data && <StockRulesForm rules={stock.data} />}
            </>
          )}
        </QueryState>
      </Stack>
    </SettingsSection>
  );
}
