import type { ProductStatus } from '@/shared/api';

export type { Product, ProductStatus, StockLabel } from '@/shared/api';

export type ProductStatusFilter = ProductStatus | 'all';
