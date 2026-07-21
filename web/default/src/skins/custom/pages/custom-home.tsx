import { createElement } from 'react'
import { Home } from '@/features/home'
import { CustomHeroCarousel } from '../components/custom-hero-carousel'

export function CustomHome() {
  return createElement(Home, {
    heroBackground: createElement(CustomHeroCarousel),
  })
}
