import { ArrowLeft2, ArrowRight2 } from "iconsax-reactjs";
import { Button } from "@/shared/ui";

export function ProductsPagination({
  page,
  pageSize,
  total,
  onPageChange,
  disabled,
}: {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  disabled?: boolean;
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  if (total === 0) return null;

  // An old link or a shrunk list can point past the end (?page=3 of 1).
  if (page > pages) {
    return (
      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        <span>На странице {page} товаров нет, всего страниц: {pages}.</span>
        <Button variant="outline" size="sm" onClick={() => onPageChange(pages)}>
          К последней странице
        </Button>
      </div>
    );
  }

  const from = (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);

  return (
    <nav
      aria-label="Страницы товаров"
      className="flex flex-wrap items-center justify-between gap-3 text-sm text-muted-foreground"
    >
      <span className="tabular-nums">
        {from}–{to} из {total}
      </span>
      {pages > 1 && (
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            aria-label="Предыдущая страница"
            disabled={disabled || page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            <ArrowLeft2 />
          </Button>
          <span className="tabular-nums text-foreground">
            {page} / {pages}
          </span>
          <Button
            variant="outline"
            size="icon"
            aria-label="Следующая страница"
            disabled={disabled || page >= pages}
            onClick={() => onPageChange(page + 1)}
          >
            <ArrowRight2 />
          </Button>
        </div>
      )}
    </nav>
  );
}
