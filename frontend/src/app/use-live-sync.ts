import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { dashboardKeys } from '@/entities/dashboard';
import { orderKeys, useOrderEvents } from '@/entities/order';
import { productKeys } from '@/entities/product';

/**
 * Keeps every signed-in page current. An order change shows in the orders, in the dashboard numbers
 * (the menu badge reads them too) and in the products, whose stock it moves.
 * It lives here because only the app layer may use several entities at once.
 */
export function useLiveSync() {
  const queryClient = useQueryClient();

  const refresh = useCallback(() => {
    for (const queryKey of [orderKeys.all, dashboardKeys.all, productKeys.all]) {
      void queryClient.invalidateQueries({ queryKey });
    }
  }, [queryClient]);

  useOrderEvents(refresh);
}
