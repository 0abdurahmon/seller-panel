import { Badge } from '@/shared/ui';
import type { ProductStatus, StockLabel } from '../model/types';

type BadgeVariant = 'success' | 'warning' | 'destructive' | 'secondary' | 'outline';

const statusMeta: Record<ProductStatus, { label: string; variant: BadgeVariant }> = {
  active: { label: 'Активен', variant: 'success' },
  draft: { label: 'Черновик', variant: 'secondary' },
  archived: { label: 'В архиве', variant: 'outline' },
};

export function ProductStatusBadge({ status }: { status: ProductStatus }) {
  const meta = statusMeta[status];
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
}

// `in_stock` has no badge: the number alone says enough.
const stockMeta: Partial<Record<StockLabel, { label: string; variant: BadgeVariant }>> = {
  low_stock: { label: 'Мало', variant: 'warning' },
  out_of_stock: { label: 'Нет в наличии', variant: 'destructive' },
};

/** The label comes from the backend (`stock_label`), so every screen shows the same one. */
export function StockLabelBadge({ label }: { label: StockLabel }) {
  const meta = stockMeta[label];
  if (!meta) return null;
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
}
