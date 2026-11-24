import { useEffect, useRef, useState } from "react";
import { CloseCircle, SearchNormal1 } from "iconsax-reactjs";
import { Input } from "@/shared/ui";

const SEARCH_DELAY_MS = 300;
/** Same limit as the `q` parameter in api/openapi.yaml. */
const MAX_LENGTH = 100;

/**
 * Search by title or SKU. Types freely and reports the trimmed text only once typing
 * pauses, so the list is not requested on every key.
 */
export const ProductSearch = ({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) => {
  const [text, setText] = useState(value);
  const [lastValue, setLastValue] = useState(value);
  const timer = useRef<number | undefined>(undefined);

  // The URL can change from outside (back button, "reset filters"): follow it, but keep
  // what is typed when it is the same search (a trailing space, for example).
  if (value !== lastValue) {
    setLastValue(value);
    if (text.trim() !== value) setText(value);
  }

  useEffect(() => () => window.clearTimeout(timer.current), []);

  function report(next: string, delay: number) {
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => onChange(next.trim()), delay);
  }

  return (
    <div className="relative w-full sm:w-72">
      <SearchNormal1 className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        type="search"
        value={text}
        maxLength={MAX_LENGTH}
        onChange={(event) => {
          setText(event.target.value);
          report(event.target.value, SEARCH_DELAY_MS);
        }}
        placeholder="Название или артикул"
        aria-label="Поиск товаров по названию или артикулу"
        className="pr-9 pl-10 [&::-webkit-search-cancel-button]:hidden"
      />
      {text && (
        <button
          type="button"
          onClick={() => {
            setText("");
            report("", 0);
          }}
          aria-label="Очистить поиск"
          className="absolute top-1/2 right-2 flex size-7 -translate-y-1/2 items-center justify-center rounded-full text-muted-foreground outline-none hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/30"
        >
          <CloseCircle className="size-4" />
        </button>
      )}
    </div>
  );
};
