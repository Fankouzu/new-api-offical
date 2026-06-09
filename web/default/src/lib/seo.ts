type SEOMeta = {
  title: string
  description: string
  robots: string
}

type RouteSEOAction =
  | { kind: 'sync'; meta: SEOMeta }
  | { kind: 'noindex'; updateTitle: false }

const SEO_FALLBACK_TITLES = new Set([
  'Page Not Found | Lizh AI',
  '页面未找到 | Lizh AI',
  'Console | Lizh AI',
  '控制台 | Lizh AI',
])

const TOPIC_META: Record<string, SEOMeta> = {
  '/use-cases/openai-compatible-api': {
    title: 'OpenAI-Compatible API for GPT, Gemini, DeepSeek and Qwen | Lizh AI',
    description:
      'Use Lizh AI as an OpenAI-compatible API gateway for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream AI models.',
    robots: 'index,follow',
  },
  '/compare/ai-api-pricing': {
    title: 'AI API Pricing Comparison for Mainstream Models | Lizh AI',
    description:
      'Compare approximate AI API prices across GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other models available in Lizh AI.',
    robots: 'index,follow',
  },
  '/providers/gemini-api': {
    title: 'Gemini API Pricing and Model Access | Lizh AI',
    description:
      'Explore Gemini API model options, approximate prices, model IDs, and OpenAI-compatible access paths through Lizh AI.',
    robots: 'index,follow',
  },
  '/providers/deepseek-api': {
    title: 'DeepSeek API Pricing and Model Access | Lizh AI',
    description:
      'Explore DeepSeek API options for reasoning, coding, and cost-conscious AI workloads with approximate prices and model IDs in Lizh AI.',
    robots: 'index,follow',
  },
  '/providers/qwen-api': {
    title: 'Qwen API Pricing and Model Access | Lizh AI',
    description:
      'Explore Qwen API pricing, model IDs, multilingual and coding use cases, and OpenAI-compatible access through Lizh AI.',
    robots: 'index,follow',
  },
  '/guides/openai-sdk-compatible': {
    title: 'OpenAI SDK Compatible API Guide for Multiple AI Models | Lizh AI',
    description:
      'Learn how to use OpenAI-compatible client patterns, API base URLs, API keys, and model IDs to access multiple AI model families from Lizh AI.',
    robots: 'index,follow',
  },
}

const DEFAULT_META: SEOMeta = {
  title: 'Lizh AI | AI Model Marketplace and OpenAI-Compatible API Gateway',
  description:
    'Lizh AI is an AI model marketplace with OpenAI-compatible API access for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream models.',
  robots: 'index,follow',
}

export function syncRouteSEO(href: string) {
  if (typeof document === 'undefined') return
  const url = new URL(href, window.location.origin)
  const action = getRouteSEOAction(url.pathname)
  if (action.kind === 'noindex') {
    upsertMeta('name', 'robots', 'noindex,nofollow')
    removeCanonical()
    restoreAppTitleIfNeeded()
    return
  }
  const meta = action.meta
  document.title = meta.title
  upsertMeta('name', 'title', meta.title)
  upsertMeta('name', 'description', meta.description)
  upsertMeta('name', 'robots', meta.robots)
  upsertCanonical(`${window.location.origin}${url.pathname}`)
}

function getRouteSEOAction(pathname: string): RouteSEOAction {
  const path = pathname.replace(/\/+$/, '') || '/'
  if (path === '/') return { kind: 'sync', meta: DEFAULT_META }
  if (TOPIC_META[path]) return { kind: 'sync', meta: TOPIC_META[path] }
  if (path === '/pricing') {
    return {
      kind: 'sync',
      meta: {
        title:
          'AI Model API Pricing Marketplace | GPT, Gemini, DeepSeek, Qwen - Lizh AI',
        description:
          'Compare AI model API prices in Lizh AI, including GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream models.',
        robots: 'index,follow',
      },
    }
  }
  if (path.startsWith('/pricing/')) {
    const modelId = safeDecodeURIComponent(path.slice('/pricing/'.length))
    return {
      kind: 'sync',
      meta: {
        title: `${formatModelName(modelId)} API Pricing | Lizh AI`,
        description: `View ${formatModelName(modelId)} API pricing, capabilities, model ID, and OpenAI-compatible access information in Lizh AI.`,
        robots: 'index,follow',
      },
    }
  }
  if (path === '/rankings') {
    return {
      kind: 'sync',
      meta: {
        title: 'Popular AI Model Rankings | Lizh AI',
        description:
          'Explore popular AI model rankings for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, and other model families available through Lizh AI.',
        robots: 'index,follow',
      },
    }
  }
  if (path === '/about') {
    return {
      kind: 'sync',
      meta: {
        title: 'About Lizh AI | AI Model Marketplace and API Gateway',
        description:
          'Learn about Lizh AI, an AI model marketplace for multi-model API access, OpenAI-compatible integration, and unified account settlement.',
        robots: 'index,follow',
      },
    }
  }
  if (path === '/privacy-policy') {
    return {
      kind: 'sync',
      meta: {
        title: 'Privacy Policy | Lizh AI',
        description:
          'Review the Lizh AI privacy policy for account, API usage, billing, and service data handling.',
        robots: 'index,follow',
      },
    }
  }
  if (path === '/user-agreement') {
    return {
      kind: 'sync',
      meta: {
        title: 'User Agreement | Lizh AI',
        description:
          'Review the Lizh AI user agreement for API gateway usage, account, billing, and compliance requirements.',
        robots: 'index,follow',
      },
    }
  }
  return { kind: 'noindex', updateTitle: false }
}

function formatModelName(modelId: string): string {
  return modelId
    .split(/[-_/\s]+/)
    .filter(Boolean)
    .map((part) => {
      const lower = part.toLowerCase()
      if (
        ['gpt', 'glm', 'api', 'json', 'vl', 'tts', 'ocr', 'ai'].includes(lower)
      ) {
        return part.toUpperCase()
      }
      if (lower === 'qwen') return 'Qwen'
      if (lower === 'deepseek') return 'DeepSeek'
      if (lower === 'gemini') return 'Gemini'
      if (lower === 'doubao') return 'Doubao'
      if (lower === 'minimax') return 'MiniMax'
      if (lower === 'kimi') return 'Kimi'
      return part.charAt(0).toUpperCase() + part.slice(1)
    })
    .join(' ')
}

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

function upsertMeta(
  attrName: 'name' | 'property',
  attrValue: string,
  content: string
) {
  const selector = `meta[${attrName}="${attrValue}"]`
  let element = document.head.querySelector<HTMLMetaElement>(selector)
  if (!element) {
    element = document.createElement('meta')
    element.setAttribute(attrName, attrValue)
    document.head.appendChild(element)
  }
  element.content = content
}

function upsertCanonical(href: string) {
  let element = document.head.querySelector<HTMLLinkElement>(
    'link[rel="canonical"]'
  )
  if (!element) {
    element = document.createElement('link')
    element.rel = 'canonical'
    document.head.appendChild(element)
  }
  element.href = href
}

function removeCanonical() {
  document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]')?.remove()
}

function restoreAppTitleIfNeeded() {
  if (!SEO_FALLBACK_TITLES.has(document.title)) return

  const systemName = getCachedSystemName()
  if (systemName) {
    document.title = systemName
  }
}

function getCachedSystemName(): string {
  try {
    const saved = window.localStorage?.getItem('status')
    if (!saved) return ''
    const status = JSON.parse(saved) as { system_name?: unknown }
    return typeof status.system_name === 'string' ? status.system_name : ''
  } catch {
    return ''
  }
}
