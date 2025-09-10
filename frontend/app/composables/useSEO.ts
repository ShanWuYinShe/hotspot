/**
 * SEO优化 Composable
 * 统一管理页面SEO元数据
 */

export interface SEOConfig {
  title?: string
  description?: string
  keywords?: string[]
  image?: string
  url?: string
  type?: 'website' | 'article' | 'profile'
  author?: string
  publishedTime?: string
  modifiedTime?: string
}

export const useSEO = (config: SEOConfig = {}) => {
  const { t } = useI18n()
  const route = useRoute()
  const runtimeConfig = useRuntimeConfig()
  
  const siteUrl = runtimeConfig.public.siteUrl
  const defaultImage = runtimeConfig.public.shareDefaultImage
  
  // 构建完整URL
  const fullUrl = config.url || `${siteUrl}${route.path}`
  
  // 设置页面元数据
  useSeoMeta({
    title: config.title || t('app.title'),
    description: config.description || t('app.description'),
    keywords: config.keywords?.join(', '),
    
    // Open Graph
    ogTitle: config.title || t('app.title'),
    ogDescription: config.description || t('app.description'),
    ogImage: config.image || defaultImage,
    ogUrl: fullUrl,
    ogType: config.type || 'website',
    ogSiteName: t('app.title'),
    
    // Twitter Card
    twitterCard: 'summary_large_image',
    twitterTitle: config.title || t('app.title'),
    twitterDescription: config.description || t('app.description'),
    twitterImage: config.image || defaultImage,
    
    // 文章特定元数据
    ...(config.type === 'article' && {
      articleAuthor: config.author ? [config.author] : undefined,
      articlePublishedTime: config.publishedTime,
      articleModifiedTime: config.modifiedTime,
    })
  })
  
  // 设置canonical URL
  useHead({
    link: [
      {
        rel: 'canonical',
        href: fullUrl
      }
    ]
  })
  
  // 结构化数据
  if (config.type === 'article') {
    useHead({
      script: [
        {
          type: 'application/ld+json',
          innerHTML: JSON.stringify({
            '@context': 'https://schema.org',
            '@type': 'Article',
            headline: config.title,
            description: config.description,
            image: config.image,
            author: {
              '@type': 'Person',
              name: config.author
            },
            publisher: {
              '@type': 'Organization',
              name: t('app.title'),
              logo: {
                '@type': 'ImageObject',
                url: `${siteUrl}/logo.png`
              }
            },
            datePublished: config.publishedTime,
            dateModified: config.modifiedTime,
            url: fullUrl
          })
        }
      ]
    })
  }
}

