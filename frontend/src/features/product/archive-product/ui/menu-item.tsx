import { ArchiveBox } from "iconsax-reactjs";
import type { Product } from "@/entities/product";
import { DropdownMenuItem } from "@/shared/ui";
import { useArchiveProduct } from "../model/dialog.store";

/** Archived products have nothing left to archive. */
export const ArchiveProductMenuItem = ({ product }: { product: Product }) => {
  const openProduct = useArchiveProduct((state) => state.openProduct);
  if (product.status === "archived") return null;

  return (
    <DropdownMenuItem variant="destructive" onClick={() => openProduct(product)}>
      <ArchiveBox />
      Архивировать
    </DropdownMenuItem>
  );
};
