import type { ThemeManifest } from './contracts'
import { resolveRuntimeRoute } from './resolve-route'
import type { SkinPageRenderPlan } from './skin-page-plan'

export function createSkinRouteRenderPlan(
  activeSkin: ThemeManifest,
  routeId: string,
  routePath: `/${string}`
): Extract<SkinPageRenderPlan, { shell: 'skin' }> {
  const component = resolveRuntimeRoute(activeSkin, routeId, routePath)

  if (!activeSkin.shell) {
    throw new Error(
      `Skin ${activeSkin.id} route ${routeId} at generated path ${routePath} requires a skin shell, but none is configured`
    )
  }

  return {
    shell: 'skin',
    definition: { component, shell: 'skin' },
    skinId: activeSkin.id,
    Shell: activeSkin.shell,
  }
}
