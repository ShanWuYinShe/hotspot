'use client'

import { Suspense, useEffect, useState, type ReactNode } from 'react'

export function MswProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (process.env.NODE_ENV !== 'development') {
      setReady(true)
      return
    }

    async function init() {
      const { worker } = await import('@/mocks/browser')
      const { initMock } = await import('@/mocks/toggle')
      await initMock(worker, process.env.NEXT_PUBLIC_API_MOCK !== 'false')
      setReady(true)
    }

    init()
  }, [])

  if (!ready) return null

  return <Suspense fallback={null}>{children}</Suspense>
}
