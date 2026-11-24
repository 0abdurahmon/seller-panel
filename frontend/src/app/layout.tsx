import { Outlet } from 'react-router';
import { SidebarProvider } from '@/shared/providers';
import { Aside, MobileMenu } from '@/widgets/aside';
import { Header } from '@/widgets/header';

export const AppLayout = () => {
  return (
    <SidebarProvider>
      <div className="flex h-dvh bg-background text-foreground">
        <Aside />
        <div className="flex h-full min-w-0 flex-1 flex-col overflow-y-auto px-5 pt-3">
          <Header leading={<MobileMenu />} />
          <main className="flex flex-1 flex-col py-6">
            <Outlet />
          </main>
        </div>
      </div>
    </SidebarProvider>
  );
};
