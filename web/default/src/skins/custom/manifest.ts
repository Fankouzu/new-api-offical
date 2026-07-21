import type { ThemeManifest } from '@/skins/runtime/contracts'
import customBuildManifest from './build-manifest'
import { CustomPublicShell } from './shell/custom-public-shell'

const customManifest = {
  id: 'custom',
  build: customBuildManifest,
  pages: {},
  routes: [],
  navigation: { header: [], footer: [] },
  shell: CustomPublicShell,
} as const satisfies ThemeManifest

export default customManifest
