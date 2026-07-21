import type {
  PublicPageKey,
  SkinPageDefinition,
  ThemeManifest,
} from './contracts'

export function resolveSkinPage(
  activeSkin: ThemeManifest,
  defaultSkin: ThemeManifest,
  page: PublicPageKey
): SkinPageDefinition {
  const resolvedPage = activeSkin.pages[page] ?? defaultSkin.pages[page]

  if (!resolvedPage) {
    throw new Error(`No page registered for ${page}`)
  }

  return resolvedPage
}
