import { CloseCircle } from "iconsax-reactjs";
import { canCancel, type OrderTarget } from "@/entities/order";
import { DropdownMenuItem } from "@/shared/ui";
import { useCancelOrder } from "../model/dialog.store";

/** Once the courier has the order, or it is final, it cannot be cancelled. */
export const CancelOrderMenuItem = ({ order }: { order: OrderTarget }) => {
  const openOrder = useCancelOrder((state) => state.openOrder);
  if (!canCancel(order.status)) return null;

  return (
    <DropdownMenuItem variant="destructive" onClick={() => openOrder(order)}>
      <CloseCircle />
      Отменить заказ
    </DropdownMenuItem>
  );
};
