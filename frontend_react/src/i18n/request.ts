import { getRequestConfig } from 'next-intl/server';
import { cookies, headers } from 'next/headers';
import { IntlErrorCode } from 'next-intl';

import { defaultLocale, locales } from './config';

const LOCALE_COOKIE = 'locale';

const resolveLocale = async () => {
  const cookieStore = await cookies();
  const cookieLocale = cookieStore.get(LOCALE_COOKIE)?.value;
  if (cookieLocale && locales.includes(cookieLocale as typeof locales[number])) {
    return cookieLocale as typeof locales[number];
  }

  const headerStore = await headers();
  const acceptLanguage = headerStore.get('accept-language') ?? '';
  const lowered = acceptLanguage.toLowerCase();
  if (lowered.includes('zh')) {
    return 'zh';
  }

  return defaultLocale;
};

export default getRequestConfig(async () => {
  const locale = await resolveLocale();
  const messages = (await import(`../../messages/${locale}.json`)).default;

  return {
    locale,
    messages,
    getMessageFallback({ namespace, key }) {
      return namespace ? `${namespace}.${key}` : key;
    },
    onError(error) {
      if (error.code !== IntlErrorCode.MISSING_MESSAGE) {
        console.error(error);
      }
    },
  };
});
