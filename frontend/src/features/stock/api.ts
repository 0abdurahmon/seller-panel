import { ApiError, apiFetch } from '../../shared/api/client'

/** Stock state of a product, decided by the backend (see D5). */
export type StockLabel = 'in_stock' | 'low_stock' | 'out_of_stock'

/** Response of PATCH /products/{id}/stock. */
export interface StockLevel {
  product_id: number
  stock: number
  reserved: number
  available: number
  stock_label: StockLabel
  updated_at: string
}

/** Largest stock the backend accepts (stock.MaxStock). */
export const MAX_STOCK = 1_000_000

/** Sets the absolute stock of a product. */
export function updateStock(productId: number, stock: number, signal?: AbortSignal): Promise<StockLevel> {
  return apiFetch<StockLevel>(`/products/${productId}/stock`, {
    method: 'PATCH',
    body: JSON.stringify({ stock }),
    signal,
  })
}

/** Turns a failed stock update into a message for the seller. */
export function stockErrorMessage(err: unknown): string {
  if (!(err instanceof ApiError)) return 'Something went wrong. Try again.'
  switch (err.code) {
    case 'stock_below_reserved':
      return 'Cannot go below the units reserved by accepted orders.'
    case 'product_archived':
      return 'Archived products cannot change stock.'
    case 'product_not_found':
      return 'This product no longer exists.'
    case 'validation_error':
      return `Enter a whole number from 0 to ${MAX_STOCK.toLocaleString()}.`
    case 'unauthorized':
      return 'Your session has expired. Log in again.'
    default:
      return err.message
  }
}
