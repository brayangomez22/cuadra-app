import { paths } from '@/app/routes/paths';
import { ButtonLink, IconPlus, IconUpload } from '@/shared/ui';

type Props = { canWrite: boolean; canImport: boolean };

/** The actions of the catalog: a new product and, for who loads the catalog, the import from a spreadsheet. */
export function ProductListActions({ canWrite, canImport }: Props) {
  return (
    <>
      {canWrite && (
        <ButtonLink to={paths.products.new()} icon={<IconPlus />}>
          Nuevo producto
        </ButtonLink>
      )}
      {canImport && (
        <ButtonLink to={paths.imports.new()} variant="secondary" icon={<IconUpload />}>
          Importar desde Excel
        </ButtonLink>
      )}
    </>
  );
}
