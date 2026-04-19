import type { NextApiRequest, NextApiResponse } from 'next';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8080';

// proxies browser requests through Next so the browser sees same-origin cookies
// while the backend is reached over an internal URL (in AWS: the internal ALB).
export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const pathParts = req.query.path;
  const path = Array.isArray(pathParts) ? pathParts.join('/') : String(pathParts ?? '');
  const search = req.url?.includes('?') ? req.url.slice(req.url.indexOf('?')) : '';
  const upstream = `${BACKEND_URL}/api/v1/${path}${search}`;

  const headers: Record<string, string> = {};
  for (const [k, v] of Object.entries(req.headers)) {
    if (typeof v === 'string' && ['cookie', 'content-type', 'accept', 'x-user-id'].includes(k.toLowerCase())) {
      headers[k] = v;
    }
  }

  let body: BodyInit | undefined;
  if (req.method && req.method !== 'GET' && req.method !== 'HEAD') {
    body = JSON.stringify(req.body ?? {});
    headers['content-type'] = 'application/json';
  }

  const upstreamRes = await fetch(upstream, {
    method: req.method,
    headers,
    body,
  });

  upstreamRes.headers.forEach((value, key) => {
    if (key.toLowerCase() === 'content-encoding') return;
    res.setHeader(key, value);
  });
  res.status(upstreamRes.status);
  const buf = Buffer.from(await upstreamRes.arrayBuffer());
  res.send(buf);
}
