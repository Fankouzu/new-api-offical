import type {
  DefaultThemeManifest,
  PublicPageKey,
  SkinPageDefinition,
  ThemeManifest,
} from './contracts'

export function resolveSkinPage(
  activeSkin: ThemeManifest,
  defaultSkin: DefaultThemeManifest,
  page: PublicPageKey
): SkinPageDefinition {
  const resolvedPage = activeSkin.pages[page] ?? defaultSkin.pages[page]

  if (!resolvedPage) {
    throw new Error(
      `No page registered for ${page} in active skin ${activeSkin.id} or default skin ${defaultSkin.id}`
    )
  }

  return resolvedPage
}
