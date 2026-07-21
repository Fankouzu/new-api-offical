import type { SkinComponent, SkinRuntimeRoute } from './contracts'

export function resolveRuntimeRoute(
  routes: readonly SkinRuntimeRoute[],
  routeId: string
): SkinComponent {
  const matches = routes.filter((route) => route.id === routeId)

  if (matches.length === 0) {
    throw new Error(`Active skin route is not registered: ${routeId}`)
  }

  if (matches.length > 1) {
    throw new Error(
      `Active skin route is registered more than once: ${routeId}`
    )
  }

  return matches[0].component
}
