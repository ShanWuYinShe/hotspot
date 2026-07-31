'use client'

import { useEffect } from 'react'
import { setApiBaseUrl } from '@/api'
import '@/styles/index.css'

export function AppInit() {
  useEffect(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? window.location.origin
    setApiBaseUrl(apiUrl)
  }, [])

  return null
}
