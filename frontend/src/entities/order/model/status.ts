import type { OrderStatus } from './types';

export const ORDER_STATUSES: OrderStatus[] = [
  'new',
  'accepted',
  'packed',
  'handed_to_courier',
  'delivered',
  'cancelled',
];

export const ORDER_STATUS_LABEL: Record<OrderStatus, string> = {
  new: 'Новый',
  accepted: 'Принят',
  packed: 'Собран',
  handed_to_courier: 'Передан курьеру',
  delivered: 'Доставлен',
  cancelled: 'Отменён',
};

// The status machine of docs/DOMAIN.md. The backend checks every change again.
const NEXT_STATUS: Partial<Record<OrderStatus, OrderStatus>> = {
  new: 'accepted',
  accepted: 'packed',
  packed: 'handed_to_courier',
  handed_to_courier: 'delivered',
};

/** The status an order moves to next, or null when it is final. */
export function getNextStatus(status: OrderStatus): OrderStatus | null {
  return NEXT_STATUS[status] ?? null;
}

/** An order can be cancelled until the courier takes it. */
export function canCancel(status: OrderStatus) {
  return status === 'new' || status === 'accepted' || status === 'packed';
}

/** Accepting reserves the goods; cancelling from these statuses gives them back. */
export function holdsStock(status: OrderStatus) {
  return status === 'accepted' || status === 'packed';
}
