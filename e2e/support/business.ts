import { randomInt, randomUUID } from 'node:crypto';

// DIAN weights for the verification digit, from the rightmost digit of the NIT.
const weights = [3, 7, 13, 17, 19, 23, 29, 37, 41, 43, 47, 53, 59, 67, 71];

/** The DIAN verification digit of a NIT (without the digit). */
export function verificationDigit(nit: string): number {
  const digits = [...nit].reverse().map(Number);
  const sum = digits.reduce((total, digit, index) => total + digit * weights[index], 0);
  const rest = sum % 11;
  return rest > 1 ? 11 - rest : rest;
}

/** A fictitious company NIT (9 digits starting with 9) with a valid verification digit. */
export function fakeNit(): string {
  const base = `9${String(randomInt(0, 100_000_000)).padStart(8, '0')}`;
  return `${base}-${verificationDigit(base)}`;
}

/** The data of a new business per test, so each one signs up its own tenant. */
export function newBusiness(label: string) {
  const id = randomUUID().slice(0, 8);
  return {
    tenantName: `${label} ${id}`,
    nit: fakeNit(),
    ownerName: 'Dueño de prueba',
    email: `e2e-${id}@example.com`,
    password: `clave-${id}-segura`,
  };
}
