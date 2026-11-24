import { Badge } from '@/shared/ui';
import { ORDER_STATUS_LABEL } from '../model/status';
import type { OrderStatus } from '../model/types';

type BadgeVariant = 'default' | 'success' | 'warning' | 'destructive' | 'secondary';

// A new order is the one that needs the seller, so it stands out.
const VARIANT: Record<OrderStatus, BadgeVariant> = {
  new: 'warning',
  accepted: 'default',
  packed: 'default',
  handed_to_courier: 'secondary',
  delivered: 'success',
  cancelled: 'destructive',
};

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  return <Badge variant={VARIANT[status]}>{ORDER_STATUS_LABEL[status]}</Badge>;
}
