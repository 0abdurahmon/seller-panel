import type { ProductStatus } from '@/shared/api/model';

export type { Product, ProductStatus, StockLabel } from '@/shared/api/model';

export type ProductStatusFilter = ProductStatus | 'all';
