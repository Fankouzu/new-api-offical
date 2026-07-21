import type { ComponentType, ReactNode } from 'react'
import type {
  DefaultThemeManifest,
  PublicPageKey,
  SkinPageDefinition,
  ThemeManifest,
} from './contracts'
import { resolveSkinPage } from './resolve-page'

type SkinPageRenderPlan =
  | {
      shell: 'self'
      definition: SkinPageDefinition
    }
  | {
      shell: 'skin'
      definition: SkinPageDefinition
      skinId: string
      Shell: ComponentType<{ children: ReactNode }>
    }

export function createSkinPageRenderPlan(
  activeSkin: ThemeManifest,
  defaultSkin: DefaultThemeManifest,
  page: PublicPageKey
): SkinPageRenderPlan {
  const definition = resolveSkinPage(activeSkin, defaultSkin, page)

  if (definition.shell === 'self') {
    return { shell: 'self', definition }
  }

  if (!activeSkin.shell) {
    throw new Error(
      `Skin ${activeSkin.id} page ${page} requires a skin shell, but none is configured`
    )
  }

  return {
    shell: 'skin',
    definition,
    skinId: activeSkin.id,
    Shell: activeSkin.shell,
  }
}
