import { createIcon } from './createIcon';

// Two people: the business's customers.
export const IconUsers = createIcon(
  <>
    <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
    <circle cx="9" cy="7" r="4" />
    <path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
  </>,
);

// A person with a key: the company's users and their access.
export const IconUserKey = createIcon(
  <>
    <circle cx="9" cy="7" r="4" />
    <path d="M2 21v-2a4 4 0 0 1 4-4h6" />
    <circle cx="17.5" cy="17.5" r="2.5" />
    <path d="M19.3 15.7 22 13m-1 1 1 1" />
  </>,
);

// A clock turning back: the record of who did what, and when.
export const IconHistory = createIcon(
  <>
    <path d="M3 12a9 9 0 1 0 3-6.7L3 8" />
    <path d="M3 3v5h5M12 7v5l3 3" />
  </>,
);
