import { cn } from 'cn';
import type { ProductStatus, StockLabel } from '../model/types';

const badgeClass = 'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap';

const statusMeta: Record<ProductStatus, { label: string; className: string }> = {
  active: {
    label: 'Активен',
    className: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300',
  },
  draft: {
    label: 'Черновик',
    className: 'bg-muted text-muted-foreground',
  },
  archived: {
    label: 'В архиве',
    className: 'bg-zinc-200 text-zinc-700 dark:bg-zinc-500/15 dark:text-zinc-300',
  },
};

export function ProductStatusBadge({ status }: { status: ProductStatus }) {
  const meta = statusMeta[status];
  return <span className={cn(badgeClass, meta.className)}>{meta.label}</span>;
}

// `in_stock` has no badge: the number alone says enough.
const stockMeta: Partial<Record<StockLabel, { label: string; className: string }>> = {
  low_stock: {
    label: 'Мало',
    className: 'bg-amber-100 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300',
  },
  out_of_stock: {
    label: 'Нет в наличии',
    className: 'bg-red-100 text-red-800 dark:bg-red-500/15 dark:text-red-300',
  },
};

/** The label comes from the backend (`stock_label`), so every screen shows the same one. */
export function StockLabelBadge({ label }: { label: StockLabel }) {
  const meta = stockMeta[label];
  if (!meta) return null;
  return <span className={cn(badgeClass, meta.className)}>{meta.label}</span>;
}
