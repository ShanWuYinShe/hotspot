import { MainNav } from '@/components/common/main-nav';

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen bg-background">
      <MainNav />
      <main className="flex-1 p-8">
        {children}
      </main>
    </div>
  );
}
