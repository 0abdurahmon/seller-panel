import { Edit2 } from "iconsax-reactjs";
import type { Product } from "@/entities/product";
import { DropdownMenuItem } from "@/shared/ui";
import { useEditProduct } from "../model/modal.store";

/** Archived products cannot be edited (the API answers 409 product_archived). */
export const EditProductMenuItem = ({ product }: { product: Product }) => {
  const openProduct = useEditProduct((state) => state.openProduct);
  if (product.status === "archived") return null;

  return (
    <DropdownMenuItem onClick={() => openProduct(product)}>
      <Edit2 />
      Редактировать
    </DropdownMenuItem>
  );
};
