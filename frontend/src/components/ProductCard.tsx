import Link from 'next/link';
import type { Product } from '@/lib/api';
import type { AssignmentMap } from '@/lib/experiments';
import { variantFor } from '@/lib/experiments';

type Props = {
  product: Product;
  assignments: AssignmentMap;
};

function formatPrice(cents: number, mode: string) {
  const dollars = cents / 100;
  if (mode === 'treatment') {
    return `$${dollars.toFixed(2)}`;
  }
  return `$${Math.round(dollars)}`;
}

function badgeClass(variantKey: string) {
  switch (variantKey) {
    case 'treatment_a': return 'badge new';
    case 'treatment_b': return 'badge hot';
    default: return 'badge none';
  }
}

export default function ProductCard({ product, assignments }: Props) {
  const titleVariant = variantFor(assignments, 'product_title_emphasis');
  const badgeVariant = variantFor(assignments, 'product_card_badge');
  const priceVariant = variantFor(assignments, 'product_price_format');

  const titleClass = titleVariant === 'treatment' ? 'card-title emphatic' : 'card-title';

  return (
    <Link href={`/product/${product.id}`} className="card" data-testid="product-card">
      <img src={product.image_url} alt={product.title} loading="lazy" />
      <div className="card-body">
        <h3 className={titleClass}>
          {product.title}
          <span className={badgeClass(badgeVariant)}>
            {badgeVariant === 'treatment_a' ? 'New' : badgeVariant === 'treatment_b' ? 'Hot' : ''}
          </span>
        </h3>
        <div className="card-price">{formatPrice(product.price_cents, priceVariant)}</div>
      </div>
    </Link>
  );
}
