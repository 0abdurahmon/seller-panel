import { useQuery } from '@tanstack/react-query';
import { getCatalog } from '@/shared/api/generated';
import type { Category } from '@/shared/api/model';

const api = getCatalog();

export const categoryKeys = { all: ['categories'] as const };

/** Categories are seeded reference data, so they are fetched once per session. */
export function useCategories() {
  return useQuery({
    queryKey: categoryKeys.all,
    queryFn: ({ signal }) => api.getCategories({ signal }),
    select: (data) => data.items,
    staleTime: Infinity,
  });
}

/** "Электроника › Телефоны" for a child, the plain name for a top-level category. */
export function categoryLabel(category: Category, all: Category[]) {
  const parent = category.parent_id === null ? undefined : all.find((c) => c.id === category.parent_id);
  return parent ? `${parent.name} › ${category.name}` : category.name;
}

/** Looks up a category name by id; '—' while categories load or for an unknown id. */
export function useCategoryName() {
  const { data } = useCategories();
  return (id: number) => data?.find((c) => c.id === id)?.name ?? '—';
}
