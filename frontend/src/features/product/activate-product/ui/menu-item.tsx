import { TickCircle } from "iconsax-reactjs";
import type { Product } from "@/entities/product";
import { DropdownMenuItem } from "@/shared/ui";

/** Shown only for drafts: publishing makes the product orderable. */
export const ActivateProductMenuItem = ({ product }: { product: Product }) => {
  if (product.status !== "draft") return null;

  return (
    // Not wired yet: will call PUT /products/{id} with status "active".
    <DropdownMenuItem onClick={() => {}}>
      <TickCircle />
      Активировать
    </DropdownMenuItem>
  );
};
