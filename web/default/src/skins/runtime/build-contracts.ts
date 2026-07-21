export type SkinRouteNavigation = {
  labelKey: string
  position: 'header' | 'footer'
  /** Nonnegative integer; lower values render first. */
  order?: number
}

export type SkinRouteBuildDefinition = {
  id: string
  path: `/${string}`
  componentImport: string
  navigation?: SkinRouteNavigation
}

export type SkinBuildManifest = {
  id: string
  routes: readonly SkinRouteBuildDefinition[]
}
