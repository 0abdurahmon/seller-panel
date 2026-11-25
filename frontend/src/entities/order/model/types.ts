import type { OrderStatus } from '@/shared/api';

export type {
  Order,
  OrderHistoryEntry,
  OrderItem,
  OrderStatus,
  OrderSummary,
} from '@/shared/api';

export type OrderStatusFilter = OrderStatus | 'all' | 'waiting';

/** What a feature needs to act on an order: both the list row and the full order have it. */
export type OrderTarget = { id: number; number: string; status: OrderStatus };
