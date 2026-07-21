import type { SkinComponent, ThemeManifest } from './contracts'

export function resolveRuntimeRoute(
  activeSkin: ThemeManifest,
  routeId: string,
  generatedRoutePath: `/${string}`
): SkinComponent {
  const context = `Skin ${activeSkin.id} route ${routeId}`
  const buildMatches = activeSkin.build.routes.filter(
    (route) => route.id === routeId
  )

  if (buildMatches.length === 0) {
    throw new Error(
      `${context} at generated path ${generatedRoutePath} is missing from the build manifest`
    )
  }

  if (buildMatches.length > 1) {
    throw new Error(
      `${context} at generated path ${generatedRoutePath} is registered more than once in the build manifest`
    )
  }

  const buildRoute = buildMatches[0]
  if (buildRoute.path !== generatedRoutePath) {
    throw new Error(
      `${context} build path mismatch for generated path ${generatedRoutePath}: actual path ${buildRoute.path}`
    )
  }

  const runtimeMatches = activeSkin.routes.filter(
    (route) => route.id === routeId
  )

  if (runtimeMatches.length === 0) {
    throw new Error(
      `${context} at generated path ${generatedRoutePath} is missing from the runtime manifest`
    )
  }

  if (runtimeMatches.length > 1) {
    throw new Error(
      `${context} at generated path ${generatedRoutePath} is registered more than once in the runtime manifest`
    )
  }

  const runtimeRoute = runtimeMatches[0]
  if (runtimeRoute.path !== generatedRoutePath) {
    throw new Error(
      `${context} runtime path mismatch for generated path ${generatedRoutePath}: actual path ${runtimeRoute.path}`
    )
  }

  return runtimeRoute.component
}
