import { useQuery } from '@tanstack/react-query';
import { getDashboard } from '@/shared/api';

const api = getDashboard();

export const dashboardKeys = { summary: ['dashboard', 'summary'] as const };

/** The dashboard numbers (MVP item 6) from GET /dashboard/summary. */
export function useDashboardSummary() {
  return useQuery({
    queryKey: dashboardKeys.summary,
    queryFn: ({ signal }) => api.getDashboardSummary({ signal }),
    // The numbers move with every order: refresh when the seller comes back to the tab.
    refetchOnWindowFocus: true,
  });
}
