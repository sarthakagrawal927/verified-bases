/**
 * Pages Functions proxy → Workers API.
 *
 * Every `/api/*` request hitting the Pages site is forwarded to the
 * `verified-bases-api` Worker via a service binding. This keeps the
 * frontend on a single origin (no CORS in prod) and lets us deploy the
 * BE independently.
 *
 * Wire-up (one-time, in Cloudflare dashboard → Pages → Settings → Functions):
 *   Service binding   →   variable: API   →   worker: verified-bases-api
 *
 * Or in `web/wrangler.jsonc` once Pages Direct Uploads use wrangler:
 *   "services": [{ "binding": "API", "service": "verified-bases-api" }]
 */
interface Env {
  API: { fetch: typeof fetch };
}

export const onRequest: PagesFunction<Env> = async ({ request, env }) => {
  // Strip nothing — the Go Worker expects `/api/...` paths and the route
  // matches `/api/[[path]]`, so the inbound URL already carries the
  // full path.
  if (!env.API) {
    return new Response(
      JSON.stringify({
        ok: false,
        error: 'api_binding_missing',
        hint: 'Add a service binding named "API" to verified-bases-api Worker.',
      }),
      { status: 502, headers: { 'Content-Type': 'application/json' } },
    );
  }
  return env.API.fetch(request);
};
