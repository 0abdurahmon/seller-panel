import { Link } from "react-router";
import { SearchStatus } from "iconsax-reactjs";
import {
  Button,
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/shared/ui";

export const NotFoundPage = () => {
  return (
    <Empty className="rounded-3xl border border-dashed border-border bg-card">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <SearchStatus className="text-primary" />
        </EmptyMedia>
        <EmptyTitle>Страница не найдена</EmptyTitle>
        <EmptyDescription>Такой страницы нет или она была удалена.</EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button nativeButton={false} render={<Link to="/" />}>
          На главную
        </Button>
      </EmptyContent>
    </Empty>
  );
};
