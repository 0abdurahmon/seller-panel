import type { StockLabel } from './api'
import './stock.css'

const TEXT: Record<StockLabel, string> = {
  in_stock: 'In stock',
  low_stock: 'Low stock',
  out_of_stock: 'Out of stock',
}

interface Props {
  label: StockLabel
  /** Units buyers can still order (stock − reserved), shown in the tooltip. */
  available?: number
}

/** Coloured badge for a product's stock state. */
export function StockBadge({ label, available }: Props) {
  const title = available === undefined ? undefined : `${available} available`
  return (
    <span className={`stock-badge stock-badge--${label}`} title={title}>
      {TEXT[label]}
    </span>
  )
}
