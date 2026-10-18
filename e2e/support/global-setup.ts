import { apiURL } from '../playwright.config';

const deadline = 90_000;

// The API is not started by Playwright: it runs in docker-compose and may still
// be migrating. Waiting here, with a clear message if it never comes up, beats
// every test timing out on its first request.
export default async function globalSetup() {
  const until = Date.now() + deadline;
  let last = '';
  while (Date.now() < until) {
    try {
      const response = await fetch(`${apiURL}/readyz`);
      if (response.ok) return;
      last = `/readyz respondió ${response.status}`;
    } catch (error) {
      last = String(error);
    }
    await new Promise((resolve) => setTimeout(resolve, 1_000));
  }
  throw new Error(`La API no está lista en ${apiURL}. Levántela con \`make e2e-up\`. (${last})`);
}
