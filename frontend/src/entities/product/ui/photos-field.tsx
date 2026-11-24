import { useEffect, useMemo } from 'react';
import { ImagePicker } from '@/shared/ui';
import { PHOTO_LIMITS, type PhotoItem } from '../model/form-schema';

/** Previews for new files; the URLs are released when the list changes or the field unmounts. */
function useFilePreviews(items: PhotoItem[]) {
  const previews = useMemo(
    () =>
      new Map(
        items.flatMap((item) => (item.file ? [[item.id, URL.createObjectURL(item.file)] as const] : [])),
      ),
    [items],
  );
  useEffect(() => () => previews.forEach((url) => URL.revokeObjectURL(url)), [previews]);
  return previews;
}

/**
 * The product's photos: current ones first, new files after them (the backend adds new photos
 * after the kept ones). The first photo is the cover.
 */
export const PhotosField = ({
  id,
  value,
  onChange,
  invalid,
}: {
  id?: string;
  value: PhotoItem[];
  onChange: (value: PhotoItem[]) => void;
  invalid?: boolean;
}) => {
  const previews = useFilePreviews(value);
  const items = value.map((item) => ({ key: item.id, src: item.url ?? previews.get(item.id) ?? '' }));

  return (
    <ImagePicker
      id={id}
      items={items}
      max={PHOTO_LIMITS.max}
      accept={PHOTO_LIMITS.types.join(',')}
      invalid={invalid}
      firstLabel="Обложка"
      onAdd={(files) => {
        const added = files
          .slice(0, PHOTO_LIMITS.max - value.length)
          .map((file) => ({ id: crypto.randomUUID(), file }));
        const kept = value.filter((item) => item.url);
        const fresh = value.filter((item) => item.file);
        onChange([...kept, ...fresh, ...added]);
      }}
      onRemove={(key) => onChange(value.filter((item) => item.id !== key))}
    />
  );
};
