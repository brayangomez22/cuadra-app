// page wraps items as the API answers a paginated list.
export function page<T>(items: T[], total = items.length) {
  return { items, total };
}

// query returns the search params of a request sent to the fake API.
export function query(request: Request) {
  return new URL(request.url).searchParams;
}

// plain turns Intl's no-break spaces ("$\u00a012.500") into spaces to compare text.
export function plain(text: string | null | undefined) {
  return text?.replace(/\u00a0/g, ' ');
}
