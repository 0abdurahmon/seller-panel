export type { Product, ProductStatus, ProductStatusFilter, StockLabel } from './model/types';
export {
  productKeys,
  useProductCounts,
  useProducts,
  type ProductSearchFilters,
} from './model/queries';
export {
  emptyProductForm,
  PHOTO_LIMITS,
  productFormSchema,
  productToForm,
  type PhotoItem,
  type ProductFormInput,
  type ProductFormValues,
} from './model/form-schema';
export { getAvailable, getStockLabel, LOW_STOCK_THRESHOLD } from './model/stock';
export { ProductStatusBadge, StockLabelBadge } from './ui/badges';
export { ProductImage } from './ui/product-image';
export { ProductForm, type CategoryOption, type ProductFormProps } from './ui/product-form';
