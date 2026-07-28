import { RouterProvider } from 'react-router';
import { router } from './routes';
import { ThemeProvider } from './contexts/ThemeContext';
import { PlatformProvider } from './contexts/PlatformContext';
import { AuthProvider } from './contexts/AuthContext';
import { AuthGate } from './components/auth/AuthGate';
import { ComposerProvider } from './contexts/ComposerContext';

export default function App() {
  return (
    <ThemeProvider>
      <PlatformProvider>
        <AuthProvider>
          <ComposerProvider>
            <RouterProvider router={router} />
            {/* Global auth overlays — cover every route. Must be inside AuthProvider. */}
            <AuthGate />
          </ComposerProvider>
        </AuthProvider>
      </PlatformProvider>
    </ThemeProvider>
  );
}
