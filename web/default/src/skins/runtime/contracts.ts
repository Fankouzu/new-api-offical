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
export type ThemeManifest = {
  id: string
  build: SkinBuildManifest
  pages: Partial<Record<PublicPageKey, SkinPageDefinition>>
  routes: readonly SkinRuntimeRoute[]
  shell?: ComponentType<{ children: ReactNode }>
}
export type ThemeManifestFor<B extends SkinBuildManifest> = Omit<
  ThemeManifest,
  'id' | 'build' | 'routes'
> & {
  id: B['id']
  build: B
  routes: RuntimeRoutesFor<B['routes']>
}
type RuntimeRoutesFor<R extends readonly unknown[]> = {
  readonly [K in keyof R]: R[K] extends {
    id: infer I
    path: infer P
  }
    ? { id: I; path: P; component: SkinComponent }
    : R[K]
}
export type DefaultThemeManifest = Omit<ThemeManifest, 'pages'> & {
  pages: Record<PublicPageKey, SkinPageDefinition>
}
