import type { PublicPageKey } from './contracts'

const PUBLIC_PAGE_PATHS: Record<PublicPageKey, `/${string}`> = {
  home: '/',
  pricing: '/pricing',
  modelDetails: '/pricing/$modelId',
  rankings: '/rankings',
  about: '/about',
  privacyPolicy: '/privacy-policy',
  userAgreement: '/user-agreement',
}

const HOST_OWNED_PATHS = new Set<string>([
  ...Object.values(PUBLIC_PAGE_PATHS),
  '/compare/ai-api-pricing',
  '/sign-in',
  '/sign-up',
  '/forgot-password',
  '/reset',
  '/user/reset',
  '/otp',
  '/setup',
  '/401',
  '/403',
  '/404',
  '/500',
  '/503',
])

const FORBIDDEN_PREFIXES = [
  '/oauth',
  '/console',
  '/dashboard',
  '/chat2link',
  '/errors',
  '/wallet',
  '/models',
  '/channels',
  '/keys',
  '/playground',
  '/profile',
  '/subscriptions',
  '/usage-logs',
  '/users',
  '/redemption-codes',
  '/system-settings',
  '/chat',
] as const

function hasAsciiControlCharacter(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const characterCode = value.charCodeAt(index)
    if (characterCode <= 0x1f || characterCode === 0x7f) {
      return true
    }
  }

  return false
}

function isCanonicalRoutePath(path: string): boolean {
  if (
    !path.startsWith('/') ||
    path.includes('//') ||
    path.includes('?') ||
    path.includes('#') ||
    path.includes('%') ||
    path.includes('\\') ||
    hasAsciiControlCharacter(path) ||
    (path !== '/' && path.endsWith('/'))
  ) {
    return false
  }

  return !path.split('/').some((segment) => segment === '.' || segment === '..')
}

export function getPublicPagePath(page: PublicPageKey): `/${string}` {
  return PUBLIC_PAGE_PATHS[page]
}

export function assertSkinRouteAllowed(path: string): void {
  if (!isCanonicalRoutePath(path)) {
    throw new Error(`Invalid skin route path: ${path}`)
  }

  if (HOST_OWNED_PATHS.has(path)) {
    throw new Error(`Skin route conflicts with a host-owned path: ${path}`)
  }

  const forbiddenPrefix = FORBIDDEN_PREFIXES.find(
    (prefix) => path === prefix || path.startsWith(`${prefix}/`)
  )

  if (forbiddenPrefix) {
    throw new Error(`Skin route conflicts with a protected path: ${path}`)
  }
}
