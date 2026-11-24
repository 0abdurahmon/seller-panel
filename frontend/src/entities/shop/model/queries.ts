import { useQuery } from '@tanstack/react-query';
import { getShop } from '@/shared/api';

const api = getShop();

export const shopKeys = { current: ['shop'] as const };

/** The signed-in seller's shop. */
export function useShop() {
  return useQuery({
    queryKey: shopKeys.current,
    queryFn: ({ signal }) => api.getShop({ signal }),
    staleTime: 5 * 60_000,
  });
}
