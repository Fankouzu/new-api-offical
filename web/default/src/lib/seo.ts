type SEOMeta = {
  title: string
  description: string
  robots: string
}

const DEFAULT_META: SEOMeta = {
  title: 'Lizh AI | GPT, Gemini, DeepSeek, and Qwen API marketplace',
  description:
    'Access GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other AI models through one OpenAI-compatible API gateway with unified billing.',
  robots: 'index,follow',
}

export function syncRouteSEO(href: string) {
  if (typeof document === 'undefined') return
  const url = new URL(href, window.location.origin)
  const meta = getRouteMeta(url.pathname)
  document.title = meta.title
  upsertMeta('name', 'title', meta.title)
  upsertMeta('name', 'description', meta.description)
  upsertMeta('name', 'robots', meta.robots)
  upsertCanonical(`${window.location.origin}${url.pathname}`)
}

function getRouteMeta(pathname: string): SEOMeta {
  const path = pathname.replace(/\/+$/, '') || '/'
  if (path === '/') return DEFAULT_META
  if (path === '/pricing') {
    return {
      title:
        'AI Model API Pricing Marketplace | GPT, Gemini, DeepSeek, Qwen - Lizh AI',
      description:
        'Compare Lizh AI model API pricing for GPT, Gemini, DeepSeek, Qwen, GLM, Doubao, MiniMax, Kimi, and 50+ models with text, image, tool, and structured-output support.',
      robots: 'index,follow',
    }
  }
  if (path.startsWith('/pricing/')) {
    const modelId = safeDecodeURIComponent(path.slice('/pricing/'.length))
    return {
      title: `${formatModelName(modelId)} API pricing | Lizh AI`,
      description: `View ${formatModelName(modelId)} API pricing, capabilities, and OpenAI-compatible access details on Lizh AI.`,
      robots: 'index,follow',
    }
  }
  if (path === '/rankings') {
    return {
      title: 'Popular AI Model Rankings | Lizh AI',
      description:
        'Explore Lizh AI model usage rankings and compare demand trends for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, and other AI models.',
      robots: 'index,follow',
    }
  }
  if (path === '/about') {
    return {
      title: 'About Lizh AI | Multi-model API marketplace',
      description:
        "Learn about Lizh AI's multi-model API marketplace, OpenAI-compatible gateway, unified billing, and developer-focused model access experience.",
      robots: 'index,follow',
    }
  }
  if (path === '/privacy-policy') {
    return {
      title: 'Privacy Policy | Lizh AI',
      description:
        'Read the Lizh AI privacy policy to understand how account, API usage, billing, and service data are processed.',
      robots: 'index,follow',
    }
  }
  if (path === '/user-agreement') {
    return {
      title: 'User Agreement | Lizh AI',
      description:
        'Read the Lizh AI user agreement covering API marketplace usage, accounts, billing, and compliance requirements.',
      robots: 'index,follow',
    }
  }
  return {
    title: utilityTitle(path),
    description:
      'This page is used for account, console, or system workflows and should not be indexed by search engines.',
    robots: 'noindex,nofollow',
  }
}

function utilityTitle(path: string): string {
  if (path === '/sign-in') return 'Sign in | Lizh AI'
  if (path === '/sign-up') return 'Sign up | Lizh AI'
  if (path.includes('reset') || path === '/forgot-password') {
    return 'Reset password | Lizh AI'
  }
  if (path.startsWith('/oauth')) return 'OAuth authorization | Lizh AI'
  if (isAuthenticatedAppPath(path)) {
    return 'Console | Lizh AI'
  }
  if (path === '/setup') return 'System setup | Lizh AI'
  if (/^\/(401|403|forbidden)$/.test(path)) return 'Access denied | Lizh AI'
  if (/^\/(500|503)$/.test(path)) return 'Service error | Lizh AI'
  return 'Page not found | Lizh AI'
}

function isAuthenticatedAppPath(path: string): boolean {
  return [
    '/_authenticated',
    '/console',
    '/usage-logs',
    '/playground',
    '/wallet',
    '/tokens',
    '/settings',
    '/user',
    '/users',
    '/channels',
    '/redemption',
    '/topup',
    '/subscription',
    '/billing',
    '/logs',
  ].some((prefix) => path === prefix || path.startsWith(`${prefix}/`))
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
