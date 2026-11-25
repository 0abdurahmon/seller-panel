import { useEffect, useMemo, useRef } from 'react';
import { GalleryAdd } from 'iconsax-reactjs';
import { LOGO_LIMITS, ShopAvatar, type Shop } from '@/entities/shop';
import { Button } from '@/shared/ui';

interface LogoFieldProps {
  id?: string;
  shop: Pick<Shop, 'name' | 'logo_url'>;
  file: File | null;
  onChange: (file: File | null) => void;
  invalid?: boolean;
}

export const LogoField = ({ id, shop, file, onChange, invalid }: LogoFieldProps) => {
  const inputRef = useRef<HTMLInputElement>(null);

  const previewUrl = useMemo(() => (file ? URL.createObjectURL(file) : null), [file]);
  useEffect(() => {
    if (!previewUrl) return;
    return () => URL.revokeObjectURL(previewUrl);
  }, [previewUrl]);

  const displayedShop = {
    name: shop.name,
    logo_url: previewUrl ?? shop.logo_url,
  };

  return (
    <div className="flex flex-col items-center gap-2">
      <Button
        id={id}
        type="button"
        variant="ghost"
        aria-invalid={invalid}
        aria-label={shop.logo_url || file ? 'Изменить логотип' : 'Загрузить логотип'}
        onClick={() => inputRef.current?.click()}
        className="group/logo relative size-28 overflow-hidden rounded-3xl p-0"
      >
        <ShopAvatar shop={displayedShop} className="size-full rounded-3xl text-2xl" />
        <span className="absolute inset-0 flex flex-col items-center justify-center gap-1 bg-overlay text-xs font-medium text-primary-foreground opacity-0 transition-opacity group-hover/logo:opacity-100 group-focus-visible/logo:opacity-100">
          <GalleryAdd className="size-5" />
          <span>{shop.logo_url || file ? 'Изменить' : 'Загрузить'}</span>
        </span>
      </Button>

      {file && (
        <Button type="button" variant="ghost" size="xs" onClick={() => onChange(null)}>
          Отменить выбор
        </Button>
      )}

      <input
        ref={inputRef}
        type="file"
        accept={LOGO_LIMITS.types.join(',')}
        hidden
        onChange={(event) => {
          const picked = event.target.files?.[0] ?? null;
          event.target.value = '';
          if (picked) onChange(picked);
        }}
      />
    </div>
  );
};
