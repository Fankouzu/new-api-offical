export type SkinRouteBuildDefinition = {
  id: string
  path: `/${string}`
  componentImport: string
  navigation?: {
    labelKey: string
    position: 'header' | 'footer'
    order?: number
  }
}

export type SkinBuildManifest = {
  id: string
  routes: readonly SkinRouteBuildDefinition[]
}
