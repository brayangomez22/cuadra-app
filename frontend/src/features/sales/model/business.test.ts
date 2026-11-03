import { aUser } from '@/test/fakeApi';
import { businessOf } from './business';

describe('businessOf', () => {
  it.each([
    {
      business: 'ferretería con todos los datos',
      contact: { tenant_address: 'Calle 45 # 12-30', tenant_city: 'Medellín', tenant_phone: '604 444 1234' },
      address: 'Calle 45 # 12-30, Medellín',
      phone: '604 444 1234',
    },
    {
      business: 'almacén de ropa solo con la ciudad',
      contact: { tenant_address: '', tenant_city: 'Bogotá D.C.', tenant_phone: '' },
      address: 'Bogotá D.C.',
      phone: '',
    },
    { business: 'sin datos', contact: {}, address: '', phone: '' },
  ])('arma la dirección en una línea ($business)', ({ contact, address, phone }) => {
    expect(businessOf({ ...aUser(), ...contact })).toEqual({
      name: 'Ferretería El Tornillo',
      nit: '900123456-8',
      address,
      phone,
    });
  });
});
