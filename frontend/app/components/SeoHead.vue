<template>
  <!-- SEO Head 组件 - 使用 Nuxt 3 的 useSeoMeta -->
</template>

<script setup lang="ts">
interface Props {
  title?: string
  description?: string
  keywords?: string[]
  image?: string
  url?: string
  type?: 'website' | 'article' | 'profile'
  author?: string
  publishedTime?: string
  modifiedTime?: string
  section?: string
  tags?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  type: 'website'
})

const route = useRoute()
const config = useRuntimeConfig()

// 构建完整的 URL
const fullUrl = computed(() => {
  if (props.url) return props.url
  const baseUrl = config.public.siteUrl || 'https://hotspot.example.com'
  return `${baseUrl}${route.path}`
})

// 构建完整的图片 URL
const fullImageUrl = computed(() => {
  if (!props.image) return undefined
  if (props.image.startsWith('http')) return props.image
  const baseUrl = config.public.siteUrl || 'https://hotspot.example.com'
  return `${baseUrl}${props.image}`
})

// 构建关键词字符串
const keywordsString = computed(() => {
  if (!props.keywords?.length) return undefined
  return props.keywords.join(', ')
})

// 构建标签字符串
const tagsString = computed(() => {
  if (!props.tags?.length) return undefined
  return props.tags.join(', ')
})

// 设置 SEO 元数据
useSeoMeta({
  // 基础元数据
  title: props.title,
  description: props.description,
  keywords: keywordsString,
  author: props.author,
  
  // Open Graph
  ogTitle: props.title,
  ogDescription: props.description,
  ogImage: fullImageUrl,
  ogUrl: fullUrl,
  ogType: props.type,
  ogSiteName: 'HotSpot',
  ogLocale: 'zh_CN',
  
  // Twitter Card
  twitterCard: 'summary_large_image',
  twitterTitle: props.title,
  twitterDescription: props.description,
  twitterImage: fullImageUrl,
  twitterSite: '@hotspot',
  twitterCreator: props.author,
  
  // 文章特定元数据
  articleAuthor: props.type === 'article' ? props.author : undefined,
  articlePublishedTime: props.type === 'article' ? props.publishedTime : undefined,
  articleModifiedTime: props.type === 'article' ? props.modifiedTime : undefined,
  articleSection: props.type === 'article' ? props.section : undefined,
  articleTag: props.type === 'article' ? tagsString : undefined,
  
  // 个人资料特定元数据
  profileFirstName: props.type === 'profile' ? props.author?.split(' ')[0] : undefined,
  profileLastName: props.type === 'profile' ? props.author?.split(' ')[1] : undefined,
  profileUsername: props.type === 'profile' ? props.author : undefined,
  
  // 技术元数据
  robots: 'index, follow',
  googlebot: 'index, follow',
  viewport: 'width=device-width, initial-scale=1',
  charset: 'utf-8'
})

// 设置结构化数据
useJsonld(() => {
  const baseSchema = {
    '@context': 'https://schema.org',
    '@type': getSchemaType(),
    name: props.title,
    description: props.description,
    url: fullUrl.value,
    image: fullImageUrl.value
  }

  if (props.type === 'article') {
    return {
      ...baseSchema,
      '@type': 'Article',
      headline: props.title,
      author: {
        '@type': 'Person',
        name: props.author
      },
      datePublished: props.publishedTime,
      dateModified: props.modifiedTime,
      articleSection: props.section,
      keywords: tagsString.value
    }
  }

  if (props.type === 'profile') {
    return {
      ...baseSchema,
      '@type': 'Person',
      name: props.author,
      url: fullUrl.value
    }
  }

  return {
    ...baseSchema,
    '@type': 'WebSite',
    publisher: {
      '@type': 'Organization',
      name: 'HotSpot',
      url: config.public.siteUrl || 'https://hotspot.example.com'
    }
  }
})

function getSchemaType() {
  switch (props.type) {
    case 'article': return 'Article'
    case 'profile': return 'Person'
    default: return 'WebSite'
  }
}
</script>
