import { useCategoryOptions } from "@/entities/category";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui";

const ALL = "all";

/** Category filter; `null` means every category. */
export const CategoryFilter = ({
  value,
  onChange,
}: {
  value: number | null;
  onChange: (value: number | null) => void;
}) => {
  const { options, status } = useCategoryOptions();
  const items = [{ value: ALL, label: "Все категории" }, ...options];

  return (
    <Select
      items={items}
      value={value === null ? ALL : String(value)}
      onValueChange={(next) => onChange(!next || next === ALL ? null : Number(next))}
      disabled={status !== "success"}
    >
      <SelectTrigger className="w-full sm:w-56" aria-label="Категория">
        <SelectValue
          placeholder={status === "error" ? "Категории не загрузились" : "Загрузка…"}
        />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
};
