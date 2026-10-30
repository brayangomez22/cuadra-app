/**
 * A Colombian phone as it is read aloud: a number of 10 digits (a mobile or a
 * landline with its area code) in groups of 3, 3 and 4. Anything else is
 * shown as it was typed.
 */
export function phoneLabel(phone: string): string {
  const match = /^(\d{3})(\d{3})(\d{4})$/.exec(phone);
  return match ? match.slice(1).join(' ') : phone;
}
