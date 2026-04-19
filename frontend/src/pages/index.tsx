import type { GetServerSideProps } from 'next';
import Layout from '@/components/Layout';
import ProductCard from '@/components/ProductCard';
import { getAssignments, listProducts, type Product } from '@/lib/api';
import { variantFor, type AssignmentMap } from '@/lib/experiments';

type Props = {
  products: Product[];
  assignments: AssignmentMap;
};

export default function Home({ products, assignments }: Props) {
  const hero = variantFor(assignments, 'landing_hero_copy');
  const ctaColor = variantFor(assignments, 'landing_cta_color');
  const gridVariant = variantFor(assignments, 'product_grid_layout');

  const heading = hero === 'treatment'
    ? 'Shoes that move with your week.'
    : 'Shop running, trail, and court shoes.';
  const sub = hero === 'treatment'
    ? 'Rotated for weekday commutes and weekend mileage.'
    : 'A curated rotation for daily training and travel.';

  const ctaStyle: React.CSSProperties = ctaColor === 'treatment'
    ? { background: 'var(--accent-alt)' }
    : {};

  const gridClass = gridVariant === 'treatment' ? 'grid compact' : 'grid';

  return (
    <Layout title="Shop">
      <section className="hero">
        <h1>{heading}</h1>
        <p>{sub}</p>
        <button className="btn" style={ctaStyle}>Start shopping</button>
      </section>
      <section className={gridClass}>
        {products.map((p) => (
          <ProductCard key={p.id} product={p} assignments={assignments} />
        ))}
      </section>
    </Layout>
  );
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ req }) => {
  const cookie = req.headers.cookie ?? '';
  const [productsRes, assignmentsRes] = await Promise.all([
    listProducts({ cookie }),
    getAssignments({ cookie }),
  ]);
  return {
    props: {
      products: productsRes.products ?? [],
      assignments: assignmentsRes.assignments ?? {},
    },
  };
};
