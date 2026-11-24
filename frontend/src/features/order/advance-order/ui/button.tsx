import { getNextStatus, type OrderTarget } from "@/entities/order";
import { Button } from "@/shared/ui";
import { ADVANCE_ACTIONS } from "../model/actions";
import { useAdvanceOrder } from "../model/dialog.store";

/** A delivered or cancelled order has no next status, so no button. */
export const AdvanceOrderButton = ({ order }: { order: OrderTarget }) => {
  const openOrder = useAdvanceOrder((state) => state.openOrder);
  const next = getNextStatus(order.status);
  const action = next ? ADVANCE_ACTIONS[next] : undefined;
  if (!action) return null;

  return (
    <Button onClick={() => openOrder(order)}>
      <action.icon />
      {action.label}
    </Button>
  );
};
