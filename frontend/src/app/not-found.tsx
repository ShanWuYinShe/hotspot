import Link from 'next/link'
import { ROUTES } from '@/lib/routes'

export default function NotFound() {
  return (
    <html>
      <body>
        <div className="flex flex-col items-center justify-center h-screen gap-4 font-sans">
          <h1 className="text-2xl font-semibold">页面未找到</h1>
          <p className="text-[#888]">你访问的页面不存在</p>
          <Link href={ROUTES.home} className="text-[#6366F1] no-underline">返回首页</Link>
        </div>
      </body>
    </html>
  )
}
