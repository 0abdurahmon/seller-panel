import { BoxAdd } from "iconsax-reactjs";
import type { Product } from "@/entities/product";
import { Button, Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui";
import { useRestockProduct } from "../model/modal.store";

export const RestockProductButton = ({ product }: { product: Product }) => {
  const openProduct = useRestockProduct((s) => s.openProduct);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="hover:bg-foreground/10 dark:hover:bg-foreground/15 [&_svg:not([class*='size-'])]:size-5"
            aria-label={`Изменить остаток: ${product.title}`}
            onClick={() => openProduct(product)}
          />
        }
      >
        <BoxAdd />
      </TooltipTrigger>
      <TooltipContent>Изменить остаток</TooltipContent>
    </Tooltip>
  );
};
