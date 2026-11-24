import { ArchiveBox } from "iconsax-reactjs";
import type { Product } from "@/entities/product";
import { DropdownMenuItem } from "@/shared/ui";

/** Archived products have nothing left to archive. */
export const ArchiveProductMenuItem = ({ product }: { product: Product }) => {
  if (product.status === "archived") return null;

  return (
    // Not wired yet: will call POST /products/{id}/archive.
    <DropdownMenuItem variant="destructive" onClick={() => {}}>
      <ArchiveBox />
      Архивировать
    </DropdownMenuItem>
  );
};
