import { lazy } from 'react'
import type { DefaultThemeManifest } from '@/skins/runtime/contracts'
import defaultBuildManifest from './build-manifest'

const Home = lazy(() =>
  import('@/features/home').then((module) => ({ default: module.Home }))
)
const Pricing = lazy(() =>
  import('@/features/pricing').then((module) => ({ default: module.Pricing }))
)
const ModelDetails = lazy(() =>
  import('@/features/pricing/components/model-details').then((module) => ({
    default: module.ModelDetails,
  }))
)
const Rankings = lazy(() =>
  import('@/features/rankings').then((module) => ({ default: module.Rankings }))
)
const About = lazy(() =>
  import('@/features/about').then((module) => ({ default: module.About }))
)
const PrivacyPolicy = lazy(() =>
  import('@/features/legal').then((module) => ({
    default: module.PrivacyPolicy,
  }))
)
const UserAgreement = lazy(() =>
  import('@/features/legal').then((module) => ({
    default: module.UserAgreement,
  }))
)

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
} as const satisfies DefaultThemeManifest

export default defaultManifest
