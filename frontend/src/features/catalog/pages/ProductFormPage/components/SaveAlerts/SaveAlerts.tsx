import { Alert } from '@/shared/ui';

type Props = {
  /** The API's message when no field shows it. */
  formError: string | null;
  /** How many variants failed after the product was saved. */
  failedVariants: number;
};

const variantsCount = (n: number) => (n === 1 ? '1 variante' : `${String(n)} variantes`);

/** What went wrong saving the product, above the form. */
export function SaveAlerts({ formError, failedVariants }: Props) {
  return (
    <>
      {formError && <Alert tone="danger">{formError}</Alert>}
      {failedVariants > 0 && (
        <Alert tone="warning" title={`El producto se guardó, pero ${variantsCount(failedVariants)} no`}>
          Corrija lo marcado en la tabla de variantes y guarde de nuevo: solo se envía lo que falta.
        </Alert>
      )}
    </>
  );
}
