import { Box } from 'iconsax-reactjs';
import { cn } from 'cn';
import type { Product } from '../model/types';

/** The product's first photo, or a box icon when it has none. */
export const ProductImage = ({
  product,
  className,
}: {
  product: Pick<Product, 'photos'>;
  className?: string;
}) => {
  const src = product.photos[0];

  return (
    <div
      className={cn(
        'flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-muted text-muted-foreground',
        className,
      )}
    >
      {src ? <img src={src} alt="" className="size-full object-cover" /> : <Box className="size-1/2" />}
    </div>
  );
};
