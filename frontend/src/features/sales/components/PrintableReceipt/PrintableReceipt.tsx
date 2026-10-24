import { Printable } from '../Printable/Printable';
import { Receipt } from '../Receipt/Receipt';

/** The sale's receipt, ready for the browser to print (see Printable). */
export function PrintableReceipt(props: Parameters<typeof Receipt>[0]) {
  return (
    <Printable>
      <Receipt {...props} />
    </Printable>
  );
}
