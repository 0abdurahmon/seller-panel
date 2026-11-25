import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Controller, useForm } from "react-hook-form";
import { dashboardKeys } from "@/entities/dashboard";
import { holdsStock, orderKeys, type OrderTarget } from "@/entities/order";
import { productKeys } from "@/entities/product";
import { getApiErrorMessage, getOrders } from "@/shared/api";
import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldError,
  FieldLabel,
  Spinner,
  Textarea,
} from "@/shared/ui";
import { useCancelOrder } from "../model/dialog.store";
import { cancelOrderSchema, type CancelOrderValues } from "../model/schema";

const api = getOrders();

export const CancelOrderDialog = () => {
  const { order, close } = useCancelOrder();
  const queryClient = useQueryClient();
  const form = useForm<CancelOrderValues>({
    resolver: zodResolver(cancelOrderSchema),
    defaultValues: { reason: "" },
  });

  const mutation = useMutation({
    meta: { announcedByServer: true },
    mutationFn: ({ target, reason }: { target: OrderTarget; reason: string }) =>
      api.postOrdersIdTransition(target.id, { to: "cancelled", reason }),
    onSuccess: async () => {
      // Cancelling gives the reserved goods back, so the products change too.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: orderKeys.all }),
        queryClient.invalidateQueries({ queryKey: productKeys.all }),
        // The menu badge counts the orders waiting for action.
        queryClient.invalidateQueries({ queryKey: dashboardKeys.all }),
      ]);
      handleClose();
    },
    // The order may have moved meanwhile (another tab, the courier): show its real status.
    onError: () => queryClient.invalidateQueries({ queryKey: orderKeys.all }),
  });

  function handleClose() {
    mutation.reset();
    form.reset();
    close();
  }

  return (
    <Dialog
      open={order !== null}
      onOpenChange={(open) => !open && !mutation.isPending && handleClose()}
    >
      <DialogContent className="max-w-md!" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Отменить заказ {order?.number}?</DialogTitle>
          <DialogDescription>
            {order && holdsStock(order.status)
              ? "Зарезервированный товар вернётся на склад. "
              : "Заказ ещё не резервировал товар. "}
            Причина сохранится в истории заказа.
          </DialogDescription>
        </DialogHeader>

        <form
          noValidate
          className="space-y-4"
          onSubmit={form.handleSubmit((values) =>
            order && mutation.mutate({ target: order, reason: values.reason }),
          )}
        >
          <Controller
            name="reason"
            control={form.control}
            render={({ field, fieldState }) => (
              <Field data-invalid={!!fieldState.error}>
                <FieldLabel htmlFor="cancel-order-reason">Причина отмены</FieldLabel>
                <Textarea
                  {...field}
                  id="cancel-order-reason"
                  aria-invalid={!!fieldState.error}
                  disabled={mutation.isPending}
                />
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />

          {mutation.error && (
            <Alert variant="destructive">
              <AlertDescription>{getApiErrorMessage(mutation.error)}</AlertDescription>
            </Alert>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              disabled={mutation.isPending}
              onClick={handleClose}
            >
              Не отменять
            </Button>
            <Button type="submit" variant="destructive" disabled={mutation.isPending}>
              {mutation.isPending && <Spinner />}
              Отменить заказ
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
