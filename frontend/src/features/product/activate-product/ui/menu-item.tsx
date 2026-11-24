import { TickCircle } from "iconsax-reactjs";
import type { Product } from "@/entities/product";
import { DropdownMenuItem } from "@/shared/ui";
import { useActivateProduct } from "../model/dialog.store";

/** Shown only for drafts: activating makes the product orderable. */
export const ActivateProductMenuItem = ({ product }: { product: Product }) => {
  const openProduct = useActivateProduct((state) => state.openProduct);
  if (product.status !== "draft") return null;

  return (
    <DropdownMenuItem onClick={() => openProduct(product)}>
      <TickCircle />
      Активировать
    </DropdownMenuItem>
  );
};
