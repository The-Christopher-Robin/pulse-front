import type { AssignmentMap } from './experiments';

export type Product = {
  id: string;
  title: string;
  description: string;
  price_cents: number;
  image_url: string;
};

const SERVER_BASE = process.env.BACKEND_URL ?? 'http://localhost:8080';
const BROWSER_BASE = process.env.NEXT_PUBLIC_BACKEND_URL ?? '/api/proxy';

function baseURL() {
  return typeof window === 'undefined' ? SERVER_BASE : BROWSER_BASE;
}

type FetchOpts = {
  userId?: string;
  cookie?: string;
  signal?: AbortSignal;
};

async function request<T>(path: string, opts: FetchOpts = {}, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (opts.userId) headers.set('X-User-Id', opts.userId);
  if (opts.cookie) headers.set('Cookie', opts.cookie);
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const res = await fetch(`${baseURL()}${path}`, {
    ...init,
    headers,
    signal: opts.signal,
    // On the server this is a same-process fetch to the Go API. In the browser
    // the client proxies through Next's API route so cookies line up.
    credentials: typeof window === 'undefined' ? 'omit' : 'include',
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  return (await res.json()) as T;
}

export async function listProducts(opts: FetchOpts = {}): Promise<{ products: Product[] }> {
  return request('/api/v1/products', opts);
}

export async function getProduct(id: string, opts: FetchOpts = {}): Promise<Product> {
  return request(`/api/v1/products/${encodeURIComponent(id)}`, opts);
}

export async function getAssignments(opts: FetchOpts = {}): Promise<{ user_id: string; assignments: AssignmentMap }> {
  return request('/api/v1/assignments', opts);
}

export async function trackEvent(body: { event_type: string; target_id?: string; properties?: Record<string, unknown> }, opts: FetchOpts = {}): Promise<void> {
  await request('/api/v1/events', opts, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
