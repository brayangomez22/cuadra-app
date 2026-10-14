import { Printable } from './Printable';
import { Receipt } from './Receipt';

/** The sale's receipt, ready for the browser to print (see Printable). */
export function PrintableReceipt(props: Parameters<typeof Receipt>[0]) {
  return (
    <Printable>
      <Receipt {...props} />
    </Printable>
  );
}
