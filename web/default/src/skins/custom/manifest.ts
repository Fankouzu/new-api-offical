import { lazy } from 'react'
import type { ThemeManifest } from '@/skins/runtime/contracts'
import customBuildManifest from './build-manifest'
import { CustomPublicShell } from './shell/custom-public-shell'

const CustomHome = lazy(() =>
  import('./pages/custom-home').then((module) => ({
    default: module.CustomHome,
  }))
)

const customManifest = {
  id: 'custom',
  build: customBuildManifest,
  pages: {
    home: { component: CustomHome, shell: 'self' },
  },
  routes: [],
  shell: CustomPublicShell,
} as const satisfies ThemeManifest

export default customManifest
