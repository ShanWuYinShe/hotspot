import { MswProvider } from '@/app/MswProvider'
import { AppInit } from '@/app/AppInit'
import { ThemeProvider } from '@/contexts/ThemeContext'
import { PlatformProvider } from '@/contexts/PlatformContext'
import { AuthProvider } from '@/contexts/AuthContext'
import { ComposerProvider } from '@/contexts/ComposerContext'
import { AuthGate } from '@/components/auth/AuthGate'

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <MswProvider>
      <AppInit />
      <ThemeProvider>
        <PlatformProvider>
          <AuthProvider>
            <ComposerProvider>
              {children}
              <AuthGate />
            </ComposerProvider>
          </AuthProvider>
        </PlatformProvider>
      </ThemeProvider>
    </MswProvider>
  )
}
