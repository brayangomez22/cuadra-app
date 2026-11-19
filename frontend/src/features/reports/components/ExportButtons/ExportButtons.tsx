import { Button, IconDownload, Inline } from '@/shared/ui';
import { useExportReport, type ExportQuery } from '../../api/export';

type Props = {
  /** The report and the filters on screen. */
  query: ExportQuery;
  disabled?: boolean;
};

/** Downloads the report on screen, every row, as Excel or CSV. */
export function ExportButtons({ query, disabled = false }: Props) {
  const exportReport = useExportReport();
  const busy = exportReport.isPending ? exportReport.variables.format : undefined;
  return (
    <Inline gap="2" wrap>
      <Button
        variant="secondary"
        icon={<IconDownload />}
        disabled={disabled}
        loading={busy === 'xlsx'}
        onClick={() => {
          exportReport.mutate({ ...query, format: 'xlsx' });
        }}
      >
        Exportar a Excel
      </Button>
      <Button
        variant="ghost"
        disabled={disabled}
        loading={busy === 'csv'}
        onClick={() => {
          exportReport.mutate({ ...query, format: 'csv' });
        }}
      >
        CSV
      </Button>
    </Inline>
  );
}
