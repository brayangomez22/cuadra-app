import { describeError } from '@/shared/api/errors';
import { Alert, Button, Stack } from '@/shared/ui';
import { useRequestRecount } from '../../../../api/stockCountReview';

type Props = {
  countId: string;
  variantId: string;
  /** The product's name, when its row is loaded; otherwise it is "el producto". */
  name: string | undefined;
  /** Runs once the recount is asked: the screen shows the line to count again. */
  onRecounted: (variantId: string) => void;
};

/**
 * Approving would leave a product without stock: it was sold or moved after
 * counting it. The way out is to count it again, from here.
 */
export function ShortStockAlert({ countId, variantId, name, onRecounted }: Props) {
  const recount = useRequestRecount(countId);
  const product = name ?? 'el producto';
  return (
    <Alert tone="danger">
      <Stack gap="3">
        <p>
          No hay stock para ajustar {product}: se vendió o salió después de contarlo. Pida recontarlo y apruebe cuando
          esté contado de nuevo.
        </p>
        {recount.isError && <p>{describeError(recount.error).message}</p>}
        <div>
          <Button
            variant="secondary"
            loading={recount.isPending}
            onClick={() => recount.mutate([variantId], { onSuccess: () => onRecounted(variantId) })}
          >
            Recontar {product}
          </Button>
        </div>
      </Stack>
    </Alert>
  );
}
