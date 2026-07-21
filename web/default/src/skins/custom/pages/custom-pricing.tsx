import { createElement } from 'react'
import { Pricing } from '@/features/pricing'
import { CustomModelHero } from '../components/custom-model-hero'

export function CustomPricing() {
  return createElement(Pricing, {
    hero: createElement(CustomModelHero),
  })
}
