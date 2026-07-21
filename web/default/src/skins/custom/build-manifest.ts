import type { SkinBuildManifest } from '@/skins/runtime/build-contracts'

const customBuildManifest = {
  id: 'custom',
  routes: [],
} as const satisfies SkinBuildManifest

export default customBuildManifest
