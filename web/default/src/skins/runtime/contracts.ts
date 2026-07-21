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
export type SkinRuntimeRoute = {
  id: string
  path: `/${string}`
  component: SkinComponent
}
export type SkinNavigationLink = {
  labelKey: string
  href: `/${string}`
  order?: number
}
export type SkinNavigationConfig = {
  header: readonly SkinNavigationLink[]
  footer: readonly SkinNavigationLink[]
}
export type ThemeManifest = {
  id: string
  build: SkinBuildManifest
  pages: Partial<Record<PublicPageKey, SkinPageDefinition>>
  routes: readonly SkinRuntimeRoute[]
  navigation?: SkinNavigationConfig
  shell?: ComponentType<{ children: ReactNode }>
}
export type DefaultThemeManifest = Omit<ThemeManifest, 'pages'> & {
  pages: Record<PublicPageKey, SkinPageDefinition>
}
