import * as React from "react"
import { GalleryAdd } from "iconsax-reactjs"
import { cn } from "cn"
import { Button } from "./button"
import { CloseIcon } from "./icons"

export type ImagePickerItem = { key: string; src: string }

/**
 * A row of image thumbnails with a remove button each, and a tile that opens the file picker.
 * It only shows and reports: the caller owns the list and checks the files.
 */
function ImagePicker({
  items,
  max,
  accept,
  onAdd,
  onRemove,
  invalid = false,
  firstLabel,
  addLabel = "Добавить",
  className,
  id,
}: {
  items: ImagePickerItem[]
  max: number
  accept: string
  onAdd: (files: File[]) => void
  onRemove: (key: string) => void
  invalid?: boolean
  /** A caption on the first image, for example "Обложка". */
  firstLabel?: string
  addLabel?: string
  className?: string
  /** Goes to the add tile, so a FieldLabel can point at it. */
  id?: string
}) {
  const inputRef = React.useRef<HTMLInputElement>(null)

  return (
    <div data-slot="image-picker" className={cn("flex flex-wrap gap-3", className)}>
      {items.map((item, index) => (
        <div
          key={item.key}
          className="group/image relative size-20 overflow-hidden rounded-2xl bg-muted"
        >
          <img src={item.src} alt="" className="size-full object-cover" />
          {index === 0 && firstLabel && (
            <span className="absolute inset-x-0 bottom-0 bg-overlay py-0.5 text-center text-xs font-medium text-primary-foreground">
              {firstLabel}
            </span>
          )}
          <Button
            variant="secondary"
            size="icon-xs"
            className="absolute top-1 right-1 rounded-full"
            aria-label="Убрать фото"
            onClick={() => onRemove(item.key)}
          >
            <CloseIcon />
          </Button>
        </div>
      ))}
      {items.length < max && (
        <Button
          id={id}
          variant="outline"
          aria-invalid={invalid}
          className="size-20 flex-col gap-1 rounded-2xl border-dashed text-xs text-muted-foreground"
          onClick={() => inputRef.current?.click()}
        >
          <GalleryAdd className="size-5" />
          {addLabel}
        </Button>
      )}
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple
        hidden
        onChange={(event) => {
          const files = Array.from(event.target.files ?? [])
          // Reset, so choosing the same file again still fires a change.
          event.target.value = ""
          if (files.length) onAdd(files)
        }}
      />
    </div>
  )
}

export { ImagePicker }
