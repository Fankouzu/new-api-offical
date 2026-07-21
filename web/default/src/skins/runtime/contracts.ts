import type { ComponentType, LazyExoticComponent, ReactNode } from 'react'
import type { SkinBuildManifest } from './build-contracts'

export type PublicPageKey =
  | 'home'
  | 'pricing'
  | 'modelDetails'
  | 'rankings'
  | 'about'
  | 'privacyPolicy'
  | 'userAgreement'

export type SkinComponent = ComponentType | LazyExoticComponent<ComponentType>
export type SkinPageDefinition = {
  component: SkinComponent
  shell: 'self' | 'skin'
}
export type SkinRuntimeRoute = { id: string; component: SkinComponent }
export type ThemeManifest = {
  id: string
  build: SkinBuildManifest
  pages: Partial<Record<PublicPageKey, SkinPageDefinition>>
  routes: readonly SkinRuntimeRoute[]
  shell?: ComponentType<{ children: ReactNode }>
}
export type DefaultThemeManifest = Omit<ThemeManifest, 'pages'> & {
  pages: Record<PublicPageKey, SkinPageDefinition>
}
