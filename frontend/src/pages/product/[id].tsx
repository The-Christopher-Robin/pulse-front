import type { GetServerSideProps } from 'next';
import Layout from '@/components/Layout';
import { getAssignments, getProduct, type Product } from '@/lib/api';
import { variantFor, type AssignmentMap } from '@/lib/experiments';

type Props = {
  product: Product;
  assignments: AssignmentMap;
};

export default function ProductPage({ product, assignments }: Props) {
  const stickyCta = variantFor(assignments, 'pdp_sticky_cta') === 'treatment';
  const shippingBadge = variantFor(assignments, 'pdp_shipping_badge') === 'treatment';
  const showReco = variantFor(assignments, 'pdp_recommendation_slot') === 'treatment';

  return (
    <Layout title={product.title}>
      <article className="pdp">
        <img src={product.image_url} alt={product.title} />
        <div>
          <h1 style={{ marginTop: 0 }}>{product.title}</h1>
          <p style={{ color: 'var(--muted)' }}>{product.description}</p>
          <div style={{ fontSize: 22, fontWeight: 600, margin: '12px 0' }}>
            ${(product.price_cents / 100).toFixed(2)}
          </div>
          {shippingBadge && (
            <div style={{ color: 'var(--accent-alt)', fontSize: 13, marginBottom: 12 }}>
              Free 2-day shipping on orders over $75.
            </div>
          )}
          <button
            className="btn success"
            data-experiment="pdp_sticky_cta"
            data-variant={stickyCta ? 'treatment' : 'control'}
          >
            Add to cart
          </button>
          {showReco && (
            <div style={{ marginTop: 24 }}>
              <h3 style={{ fontSize: 14, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--muted)' }}>
                You may also like
              </h3>
              <div style={{ color: 'var(--muted)', fontSize: 14 }}>
                Personalised picks appear here once you browse a few items.
              </div>
            </div>
          )}
        </div>
      </article>
      {stickyCta && <div className="sticky-cta">Quick buy: {product.title} for ${(product.price_cents / 100).toFixed(2)}</div>}
    </Layout>
  );
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ params, req }) => {
  const id = String(params?.id ?? '');
  const cookie = req.headers.cookie ?? '';
  try {
    const [product, assignmentsRes] = await Promise.all([
      getProduct(id, { cookie }),
      getAssignments({ cookie }),
    ]);
    return { props: { product, assignments: assignmentsRes.assignments ?? {} } };
  } catch {
    return { notFound: true };
  }
};
