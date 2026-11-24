export type { Product, ProductStatus, ProductStatusFilter, StockLabel } from './model/types';
export {
  productKeys,
  useProductCounts,
  useProducts,
  type ProductSearchFilters,
} from './model/queries';
export {
  emptyProductForm,
  productFormSchema,
  productToForm,
  type ProductFormInput,
  type ProductFormValues,
} from './model/form-schema';
export { ProductStatusBadge, StockLabelBadge } from './ui/badges';
export { ProductForm, type CategoryOption, type ProductFormProps } from './ui/product-form';
