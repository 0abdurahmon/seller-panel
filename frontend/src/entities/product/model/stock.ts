import type { Product, StockLabel } from './types';

/**
 * Same threshold as the backend's default LOW_STOCK_THRESHOLD (docs/DECISIONS.md, D5).
 * A product response no longer carries `available` or `stock_label` (D13): screens derive them.
 */
export const LOW_STOCK_THRESHOLD = 5;

export function getAvailable(product: Pick<Product, 'stock' | 'reserved'>) {
  return product.stock - product.reserved;
}

export function getStockLabel(product: Pick<Product, 'stock' | 'reserved'>): StockLabel {
  const available = getAvailable(product);
  if (available <= 0) return 'out_of_stock';
  if (available <= LOW_STOCK_THRESHOLD) return 'low_stock';
  return 'in_stock';
}
