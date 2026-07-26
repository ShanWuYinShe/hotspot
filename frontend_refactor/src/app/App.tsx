import { RouterProvider } from 'react-router';
import { router } from './routes';
import { ThemeProvider } from './contexts/ThemeContext';
import { PlatformProvider } from './contexts/PlatformContext';
import { SocialProvider } from './contexts/SocialContext';
import { ComposerProvider } from './contexts/ComposerContext';

export default function App() {
  return (
    <ThemeProvider>
      <PlatformProvider>
        <SocialProvider>
          <ComposerProvider>
            <RouterProvider router={router} />
          </ComposerProvider>
        </SocialProvider>
      </PlatformProvider>
    </ThemeProvider>
  );
}