# Stock (frontend)

Owner: @Nuriddin1999. Rules and API: `docs/DECISIONS.md` → D5.

Two components for the product list. The list itself belongs to the Products feature; it renders these in its rows.

| Component | What it shows |
|-----------|---------------|
| `StockBadge` | *In stock* / *Low stock* / *Out of stock*, from the backend's `stock_label` |
| `StockEditor` | Quick stock change in the row: type a number or use −/+, save with Enter or **Save**, cancel with Escape. Sends `PATCH /products/{id}/stock` |

## Use in the product list

```tsx
import { StockBadge, StockEditor, type StockLevel } from '../stock'

// inside a row, for product p
<StockBadge label={p.stock_label} available={p.available} />
<StockEditor
  productId={p.id}
  stock={p.stock}
  reserved={p.reserved}
  disabled={p.status === 'archived'}
  onSaved={(level: StockLevel) => updateRow(p.id, level)}
/>
```

`onSaved` gets `{stock, reserved, available, stock_label}`: copy them into the row so the badge updates without reloading the list. The product list response should carry the same `stock`, `reserved`, `available` and `stock_label` fields (the backend computes the label with `stock.LabelFor`).

## Behaviour

- Nothing is sent until the seller saves, so pressing + five times is one request.
- The value cannot go below `reserved` (units promised to accepted orders) or above 1 000 000; the editor explains why instead of sending a request that will fail.
- Server errors (`stock_below_reserved`, `product_archived`, `product_not_found`, network) are shown under the input in plain words.
- When the row receives a new `stock` from outside (list reload, an order accepted), the editor drops its draft and shows the new value.
