import { useQuery } from '@tanstack/react-query';
import type { DashboardSummary } from '@/shared/api';
import { MOCK_SUMMARY } from './mock-summary';

const MOCK_DELAY_MS = 700;

export const dashboardKeys = { summary: ['dashboard', 'summary'] as const };

/**
 * The dashboard numbers (MVP item 6). For now it answers with mock data;
 * with the API it becomes `queryFn: ({ signal }) => getDashboard().getDashboardSummary({ signal })`.
 */
export function useDashboardSummary() {
  return useQuery({
    queryKey: dashboardKeys.summary,
    // The delay plays the part of the network, so the loading skeleton shows like it will with the API.
    queryFn: () => new Promise<DashboardSummary>((resolve) => setTimeout(() => resolve(MOCK_SUMMARY), MOCK_DELAY_MS)),
    // The numbers move with every order: refresh when the seller comes back to the tab.
    refetchOnWindowFocus: true,
  });
}
