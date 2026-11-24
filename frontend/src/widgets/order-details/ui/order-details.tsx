import type { ReactNode } from "react";
import {
  ORDER_STATUS_LABEL,
  OrderStatusBadge,
  useOrder,
  type Order,
  type OrderHistoryEntry,
} from "@/entities/order";
import { AdvanceOrderButton } from "@/features/order/advance-order";
import { CancelOrderButton } from "@/features/order/cancel-order";
import { getApiErrorMessage } from "@/shared/api";
import { formatDateTime, formatPrice } from "@/shared/lib";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui";

function Detail({ label, children, className }: { label: string; children: ReactNode; className?: string }) {
  return (
    <div className={className}>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="text-sm font-medium">{children}</dd>
    </div>
  );
}

function OrderItems({ order }: { order: Order }) {
  return (
    <div className="overflow-hidden rounded-2xl border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="pl-4">Товар</TableHead>
            <TableHead className="text-right">Кол-во</TableHead>
            <TableHead className="text-right">Цена</TableHead>
            <TableHead className="pr-4 text-right">Сумма</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {order.items.map((item) => (
            <TableRow key={item.id}>
              <TableCell className="pl-4">{item.title}</TableCell>
              <TableCell className="text-right tabular-nums">{item.quantity}</TableCell>
              <TableCell className="text-right whitespace-nowrap tabular-nums">
                {formatPrice(item.price)}
              </TableCell>
              <TableCell className="pr-4 text-right whitespace-nowrap tabular-nums">
                {formatPrice(item.price * item.quantity)}
              </TableCell>
            </TableRow>
          ))}
          <TableRow>
            <TableCell colSpan={3} className="pl-4 text-right font-medium">
              Итого
            </TableCell>
            <TableCell className="pr-4 text-right font-medium whitespace-nowrap tabular-nums">
              {formatPrice(order.total)}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}

function HistoryItem({ entry }: { entry: OrderHistoryEntry }) {
  return (
    <li className="relative">
      <span className="absolute -left-[21px] top-1.5 size-2 rounded-full bg-primary" />
      <p className="text-sm font-medium">
        {entry.from_status === null ? "Заказ создан" : ORDER_STATUS_LABEL[entry.to_status]}
      </p>
      <p className="text-xs text-muted-foreground">{formatDateTime(entry.created_at)}</p>
      {entry.reason && <p className="mt-1 text-sm">Причина: {entry.reason}</p>}
    </li>
  );
}

function OrderHistory({ history }: { history: OrderHistoryEntry[] }) {
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium">История</h3>
      {history.length === 0 ? (
        <p className="text-sm text-muted-foreground">Изменений статуса пока не было.</p>
      ) : (
        <ol className="ml-2 space-y-4 border-l border-border pl-4">
          {history.map((entry, index) => (
            <HistoryItem key={index} entry={entry} />
          ))}
        </ol>
      )}
    </div>
  );
}

function OrderView({ order }: { order: Order }) {
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-2">
        <OrderStatusBadge status={order.status} />
        <span className="text-xs text-muted-foreground">
          Статус изменён {formatDateTime(order.status_changed_at)}
        </span>
      </div>

      {order.status === "cancelled" && order.cancel_reason && (
        <Alert variant="destructive">
          <AlertTitle>Заказ отменён</AlertTitle>
          <AlertDescription>Причина: {order.cancel_reason}</AlertDescription>
        </Alert>
      )}

      <dl className="grid grid-cols-2 gap-4 rounded-2xl bg-muted/50 p-4">
        <Detail label="Покупатель">{order.buyer_name}</Detail>
        <Detail label="Телефон">
          <span className="tabular-nums">{order.buyer_phone}</span>
        </Detail>
        <Detail label="Адрес доставки" className="col-span-2">
          {order.delivery_address}
        </Detail>
      </dl>

      <OrderItems order={order} />
      <OrderHistory history={order.history} />

      <div className="flex flex-wrap justify-end gap-2">
        <CancelOrderButton order={order} />
        <AdvanceOrderButton order={order} />
      </div>
    </div>
  );
}

export const OrderDetails = ({
  orderId,
  onClose,
}: {
  /** The order shown in the dialog; null when it is closed. */
  orderId: number | null;
  onClose: () => void;
}) => {
  const query = useOrder(orderId);
  const order = query.data;

  return (
    <Dialog open={orderId !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] max-w-2xl! overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{order ? `Заказ ${order.number}` : "Заказ"}</DialogTitle>
          <DialogDescription>
            {order ? `Создан ${formatDateTime(order.created_at)}` : "Данные заказа"}
          </DialogDescription>
        </DialogHeader>

        {order ? (
          <OrderView order={order} />
        ) : query.isError ? (
          <Alert variant="destructive">
            <AlertTitle>Не удалось загрузить заказ</AlertTitle>
            <AlertDescription>{getApiErrorMessage(query.error)}</AlertDescription>
            <AlertAction>
              <Button size="sm" variant="outline" onClick={() => query.refetch()}>
                Повторить
              </Button>
            </AlertAction>
          </Alert>
        ) : (
          <div className="flex min-h-48 items-center justify-center">
            <Spinner className="size-6 text-primary" aria-label="Загрузка заказа" />
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};
