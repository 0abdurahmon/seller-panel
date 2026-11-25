import { useQuery } from '@tanstack/react-query';
import { MOCK_SUMMARY } from './mock-summary';

export const dashboardKeys = { summary: ['dashboard', 'summary'] as const };

/**
 * The dashboard numbers (MVP item 6). For now it answers with mock data;
 * with the API it becomes `queryFn: ({ signal }) => getDashboard().getDashboardSummary({ signal })`.
 */
export function useDashboardSummary() {
  return useQuery({
    queryKey: dashboardKeys.summary,
    queryFn: async () => MOCK_SUMMARY,
    // The numbers move with every order: refresh when the seller comes back to the tab.
    refetchOnWindowFocus: true,
  });
}
