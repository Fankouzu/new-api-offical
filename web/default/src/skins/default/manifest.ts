import type { ThemeManifest } from '@/skins/runtime/contracts'
import { About } from '@/features/about'
import { Home } from '@/features/home'
import { PrivacyPolicy, UserAgreement } from '@/features/legal'
import { Pricing } from '@/features/pricing'
import { ModelDetails } from '@/features/pricing/components/model-details'
import { Rankings } from '@/features/rankings'
import defaultBuildManifest from './build-manifest'

const defaultManifest = {
  id: 'default',
  build: defaultBuildManifest,
  pages: {
    home: { component: Home, shell: 'self' },
    pricing: { component: Pricing, shell: 'self' },
    modelDetails: { component: ModelDetails, shell: 'self' },
    rankings: { component: Rankings, shell: 'self' },
    about: { component: About, shell: 'self' },
    privacyPolicy: { component: PrivacyPolicy, shell: 'self' },
    userAgreement: { component: UserAgreement, shell: 'self' },
  },
  routes: [],
} as const satisfies ThemeManifest

export default defaultManifest
