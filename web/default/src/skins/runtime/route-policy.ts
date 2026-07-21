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

export const HOST_OWNED_PATHS = new Set<string>([
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

export const FORBIDDEN_PREFIXES = [
  '/oauth',
  '/console',
  '/dashboard',
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

export function getPublicPagePath(page: PublicPageKey): `/${string}` {
  return PUBLIC_PAGE_PATHS[page]
}

export function assertSkinRouteAllowed(path: string): void {
  if (
    !path.startsWith('/') ||
    path.includes('//') ||
    path.includes('?') ||
    path.includes('#')
  ) {
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
