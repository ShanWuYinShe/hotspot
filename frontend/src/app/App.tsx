import { RouterProvider } from 'react-router';
import { router } from './routes';
import { ThemeProvider } from './contexts/ThemeContext';
import { PlatformProvider } from './contexts/PlatformContext';
import { AuthProvider } from './contexts/AuthContext';

export default function App() {
  return (
    <ThemeProvider>
      <PlatformProvider>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </PlatformProvider>
    </ThemeProvider>
  );
}