import { useState } from 'react';
import { findByCode, useSearchProductsNow, type Sellable } from '@/features/catalog';
import { Alert, BarcodeScanner, Button, Dialog, IconCamera } from '@/shared/ui';

type Props = {
  /** The variant (or unit) whose code was read. */
  onFound: (item: Sellable) => void;
  /** Back to the form, where the quantity is next. */
  onClose: () => void;
};

// Browsers expose the camera only over HTTPS (or localhost).
const hasCamera = () => 'mediaDevices' in navigator;

/**
 * The camera of the phone as the barcode reader: it reads one code, picks its
 * product and closes, so the person types how many there are.
 */
export function ScanToCount({ onFound, onClose }: Props) {
  const [open, setOpen] = useState(false);
  const [missing, setMissing] = useState<string>();
  const searchNow = useSearchProductsNow();
  if (!hasCamera()) return null;

  const scanned = async (code: string) => {
    const found = findByCode(await searchNow(code), code);
    setOpen(false);
    setMissing(found ? undefined : code);
    if (found) onFound(found);
  };

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          onClose();
        }}
        trigger={
          <Button variant="secondary" icon={<IconCamera />}>
            Escanear con la cámara
          </Button>
        }
        title="Escanear con la cámara"
        description="Apunte al código de barras del producto que va a contar."
        size="md"
      >
        <BarcodeScanner onScan={(code) => void scanned(code)} />
      </Dialog>
      {missing && <Alert tone="warning">No encontramos «{missing}» entre los productos activos.</Alert>}
    </>
  );
}
