import type { Me } from '@/features/auth';

/** Who sells, as the receipt and the cash closing print it under the name. */
export type Business = {
  name: string;
  nit: string;
  /** The address and the city in one line; empty if the business has not registered them. */
  address: string;
  phone: string;
};

const noBusiness: Business = { name: '', nit: '', address: '', phone: '' };

/** The business of the signed-in user, from /me. */
export function businessOf(user: Me | undefined): Business {
  if (!user) return noBusiness;
  return {
    name: user.tenant_name,
    nit: user.tenant_nit,
    address: [user.tenant_address, user.tenant_city].filter(Boolean).join(', '),
    phone: user.tenant_phone,
  };
}
