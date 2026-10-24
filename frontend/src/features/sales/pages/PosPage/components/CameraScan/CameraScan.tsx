import { useState } from 'react';
import { BarcodeScanner, Dialog, IconButton, IconCamera } from '@/shared/ui';
import type { SearchOutcome } from '../../../../model/searchText';

type Props = {
  /** Searches the code read, as Enter does in the search. */
  onScan: (code: string) => Promise<SearchOutcome['type'] | undefined>;
  /** Back to the search, where the cashier sees why the code did not go to the sale. */
  onClose: () => void;
};

// Browsers expose the camera only over HTTPS (or localhost): without it, the
// counter keeps the reader and the keyboard.
const hasCamera = () => 'mediaDevices' in navigator;

/**
 * The camera of a phone or tablet as the barcode reader. It stays open to
 * scan one product after another; a code that is not found, or that matches
 * several products, closes it so the cashier sees the search.
 */
export function CameraScan({ onScan, onClose }: Props) {
  const [open, setOpen] = useState(false);
  if (!hasCamera()) return null;

  const scanned = async (code: string) => {
    if ((await onScan(code)) !== 'add') setOpen(false);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={setOpen}
      onCloseAutoFocus={(event) => {
        event.preventDefault();
        onClose();
      }}
      trigger={<IconButton label="Escanear con la cámara" icon={<IconCamera />} variant="secondary" />}
      title="Escanear con la cámara"
      description="Apunte al código de barras o QR de cada producto: se agrega a la venta al leerlo."
      size="md"
    >
      <BarcodeScanner onScan={(code) => void scanned(code)} />
    </Dialog>
  );
}
