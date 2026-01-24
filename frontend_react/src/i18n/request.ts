import { getRequestConfig } from 'next-intl/server';
import { cookies } from 'next/headers';

import { defaultLocale, type Locale, locales } from './config';

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const localeCookie = cookieStore.get('locale')?.value;
  let locale: Locale = defaultLocale;

  if (localeCookie) {
    const normalized =
      localeCookie === 'zh-CN' ? 'zh' : localeCookie === 'en-US' ? 'en' : localeCookie;
    if (locales.includes(normalized as Locale)) {
      locale = normalized as Locale;
    }
  }

  return {
    locale,
    messages: (await import(`../../messages/${locale}.json`)).default,
  };
});
