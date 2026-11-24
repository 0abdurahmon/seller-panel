import { CloseCircle } from "iconsax-reactjs";
import { canCancel, type OrderTarget } from "@/entities/order";
import { Button } from "@/shared/ui";
import { useCancelOrder } from "../model/dialog.store";

/** Once the courier has the order, or it is final, it cannot be cancelled. */
export const CancelOrderButton = ({ order }: { order: OrderTarget }) => {
  const openOrder = useCancelOrder((state) => state.openOrder);
  if (!canCancel(order.status)) return null;

  return (
    <Button variant="destructive" onClick={() => openOrder(order)}>
      <CloseCircle />
      Отменить заказ
    </Button>
  );
};
