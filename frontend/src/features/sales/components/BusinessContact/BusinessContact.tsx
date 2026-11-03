import type { Business } from '../../model/business';

/**
 * The address and phone under the business's name on the printed documents
 * (receipt, cash closing); nothing for what the business has not registered.
 * The document's header styles the paragraphs.
 */
export function BusinessContact({ business }: { business: Business }) {
  return (
    <>
      {business.address && <p>{business.address}</p>}
      {business.phone && <p>Tel. {business.phone}</p>}
    </>
  );
}
