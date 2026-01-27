import { AdminNav } from '@/components/common/admin-nav';
import { AdminGuard } from '@/components/common/admin-guard';

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-background">
      <AdminNav />
      <main className="flex-1 p-8">
        <AdminGuard>{children}</AdminGuard>
      </main>
    </div>
  );
}
