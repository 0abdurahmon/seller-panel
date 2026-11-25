import { useMemo } from "react";
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from "@tanstack/react-table";
import { ReceiptText } from "iconsax-reactjs";
import { Skeleton } from "skullet/react";
import {
  canCancel,
  getNextStatus,
  OrderStatusBadge,
  type OrderStatusFilter,
  type OrderSummary,
} from "@/entities/order";
import { AdvanceOrderMenuItem } from "@/features/order/advance-order";
import { CancelOrderMenuItem } from "@/features/order/cancel-order";
import { formatDateTime, formatPrice } from "@/shared/lib";
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  MoreVerticalIcon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui";

function OrdersEmptyState({ status }: { status: OrderStatusFilter }) {
  const filtered = status !== "all";

  return (
    <Empty className="flex-1 h-full min-h-0 w-full rounded-3xl border border-dashed border-border bg-card p-12">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <ReceiptText className="size-5 text-primary" />
        </EmptyMedia>
        <EmptyTitle>{filtered ? "Нет заказов в этом статусе" : "Заказов пока нет"}</EmptyTitle>
        <EmptyDescription>
          {filtered
            ? "Выберите другую вкладку."
            : "Новые заказы появятся здесь, когда покупатели их оформят."}
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

function OrderActions({ order }: { order: OrderSummary }) {
  // A delivered or cancelled order has no next status and nothing left to do.
  if (!getNextStatus(order.status)) return null;

  return (
    // Clicks inside the menu (it is portaled, but React events still bubble) must not open the row's details.
    <div className="flex justify-end" onClick={(event) => event.stopPropagation()}>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button variant="ghost" size="icon-sm" aria-label={`Действия: заказ ${order.number}`} />
          }
        >
          <MoreVerticalIcon />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <AdvanceOrderMenuItem order={order} />
          {canCancel(order.status) && <DropdownMenuSeparator />}
          <CancelOrderMenuItem order={order} />
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

const columnHelper = createColumnHelper<OrderSummary>();

function buildColumns(onOpenOrder: (order: OrderSummary) => void) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- column value types differ per column
  const columns: ColumnDef<OrderSummary, any>[] = [
    columnHelper.accessor("number", {
      id: "number",
      header: "Заказ",
      cell: ({ row }) => (
        // A button so the details open from the keyboard too; a mouse click anywhere on the row does the same.
        <Button
          variant="link"
          onClick={() => onOpenOrder(row.original)}
          className="h-auto p-0 text-foreground tabular-nums hover:text-primary hover:no-underline"
        >
          {row.original.number}
        </Button>
      ),
    }),
    columnHelper.accessor("buyer_name", {
      id: "buyer",
      header: "Покупатель",
      cell: ({ getValue }) => <span className="whitespace-nowrap">{getValue()}</span>,
    }),
    columnHelper.accessor("created_at", {
      id: "created_at",
      header: "Создан",
      cell: ({ getValue }) => (
        <span className="text-sm text-muted-foreground whitespace-nowrap">
          {formatDateTime(getValue())}
        </span>
      ),
    }),
    columnHelper.accessor("total", {
      id: "total",
      header: () => <div className="text-right">Сумма</div>,
      cell: ({ getValue }) => (
        <div className="text-right whitespace-nowrap tabular-nums">{formatPrice(getValue())}</div>
      ),
    }),
    columnHelper.accessor("status", {
      id: "status",
      header: "Статус",
      cell: ({ getValue }) => <OrderStatusBadge status={getValue()} />,
    }),
    columnHelper.display({
      id: "actions",
      header: () => <span className="sr-only">Действия</span>,
      cell: ({ row }) => <OrderActions order={row.original} />,
    }),
  ];
  return columns;
}

// Stands in for real rows while loading, so skullet can measure the table's shape.
// Lengths are close to real data on purpose.
const SAMPLE_ORDERS: OrderSummary[] = Array.from({ length: 5 }, (_, i) => ({
  id: -1 - i,
  number: "#100012",
  buyer_name: "Дилноза Юсупова",
  created_at: "2026-11-24T09:12:00Z",
  total: 4_890_000,
  status: "new",
}));

function OrdersTableView({
  orders,
  onOpenOrder,
}: {
  orders: OrderSummary[];
  onOpenOrder: (order: OrderSummary) => void;
}) {
  const columns = useMemo(() => buildColumns(onOpenOrder), [onOpenOrder]);
  const table = useReactTable({ data: orders, columns, getCoreRowModel: getCoreRowModel() });

  return (
    <div className="overflow-hidden rounded-3xl border border-border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id} className="first:pl-5 last:pr-5">
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.map((row) => (
            <TableRow
              key={row.id}
              className="cursor-pointer"
              onClick={() => onOpenOrder(row.original)}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id} className="first:pl-5 last:pr-5">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export function OrdersTable({
  orders,
  isLoading,
  status,
  onOpenOrder,
}: {
  orders: OrderSummary[] | undefined;
  isLoading?: boolean;
  /** The tab on screen, so the empty state says what is empty. */
  status: OrderStatusFilter;
  onOpenOrder: (order: OrderSummary) => void;
}) {
  const isEmpty = !isLoading && orders?.length === 0;

  return (
    <Skeleton
      id="orders-table"
      // Skullet wraps the content in a pane: when empty, both stretch so the empty state
      // fills the page; with rows the table keeps its own height.
      className={isEmpty ? "flex flex-1 flex-col *:flex *:flex-1 *:flex-col" : undefined}
      label="Загрузка заказов"
      data={isLoading ? null : orders}
      sample={SAMPLE_ORDERS}
      empty={isEmpty}
      emptyMessage={<OrdersEmptyState status={status} />}
    >
      {(rows) => <OrdersTableView orders={rows} onOpenOrder={onOpenOrder} />}
    </Skeleton>
  );
}
