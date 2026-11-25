import { useMutation, useQueryClient } from "@tanstack/react-query";
import { dashboardKeys } from "@/entities/dashboard";
import { getNextStatus, orderKeys, type OrderTarget } from "@/entities/order";
import { productKeys } from "@/entities/product";
import { getApiErrorMessage, getOrders } from "@/shared/api";
import { ConfirmDialog } from "@/shared/ui";
import { ADVANCE_ACTIONS } from "../model/actions";
import { useAdvanceOrder } from "../model/dialog.store";

const api = getOrders();

export const AdvanceOrderDialog = () => {
  const { order, close } = useAdvanceOrder();
  const queryClient = useQueryClient();

  const next = order ? getNextStatus(order.status) : null;
  const action = next ? ADVANCE_ACTIONS[next] : undefined;

  const mutation = useMutation({
    meta: { announcedByServer: true },
    mutationFn: ({ target, to }: { target: OrderTarget; to: NonNullable<typeof next> }) =>
      api.postOrdersIdTransition(target.id, { to }),
    onSuccess: async () => {
      // Accepting and delivering change the stock of the products too.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: orderKeys.all }),
        queryClient.invalidateQueries({ queryKey: productKeys.all }),
        // The menu badge counts the orders waiting for action.
        queryClient.invalidateQueries({ queryKey: dashboardKeys.all }),
      ]);
      close();
    },
    // The order may have moved meanwhile (another tab, the courier): show its real status.
    onError: () => queryClient.invalidateQueries({ queryKey: orderKeys.all }),
  });

  return (
    <ConfirmDialog
      open={order !== null && action !== undefined}
      onOpenChange={(open) => {
        if (open) return;
        mutation.reset();
        close();
      }}
      title={order && action ? action.title(order.number) : ""}
      description={action?.description}
      confirmLabel={action?.confirmLabel ?? ""}
      pending={mutation.isPending}
      error={mutation.error ? getApiErrorMessage(mutation.error) : null}
      onConfirm={() => order && next && mutation.mutate({ target: order, to: next })}
    />
  );
};
