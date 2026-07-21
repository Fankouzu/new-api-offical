import type { SkinBuildManifest } from '@/skins/runtime/build-contracts'

const defaultBuildManifest = {
  id: 'default',
  routes: [],
} as const satisfies SkinBuildManifest

export default defaultBuildManifest
