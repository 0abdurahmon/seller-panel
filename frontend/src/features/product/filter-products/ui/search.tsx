import { useEffect, useRef, useState } from "react";
import { cn } from "cn";
import { SearchNormal1 } from "iconsax-reactjs";
import {
  Button,
  CloseIcon,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/shared/ui";

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
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  className?: string;
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
    <InputGroup className={cn("sm:w-72", className)}>
      <InputGroupAddon>
        <SearchNormal1 />
      </InputGroupAddon>
      <InputGroupInput
        type="text"
        inputMode="search"
        enterKeyHint="search"
        value={text}
        maxLength={MAX_LENGTH}
        onChange={(event) => {
          setText(event.target.value);
          report(event.target.value, SEARCH_DELAY_MS);
        }}
        placeholder="Название или артикул"
        aria-label="Поиск товаров по названию или артикулу"
      />
      {text && (
        <InputGroupAddon align="end">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => {
              setText("");
              report("", 0);
            }}
            aria-label="Очистить поиск"
            className="rounded-full text-muted-foreground"
          >
            <CloseIcon />
          </Button>
        </InputGroupAddon>
      )}
    </InputGroup>
  );
};
