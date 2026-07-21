import { createElement } from 'react'
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import {
  CUSTOM_HERO_IMAGES,
  CustomHeroCarousel,
  getNextHeroSlideIndex,
} from './custom-hero-carousel'

describe('CustomHeroCarousel', () => {
  it('renders the four approved images as decorative cover slides', () => {
    const markup = renderToStaticMarkup(createElement(CustomHeroCarousel))

    assert.equal(CUSTOM_HERO_IMAGES.length, 4)
    for (const image of CUSTOM_HERO_IMAGES) {
      assert.match(markup, new RegExp(image.replaceAll('/', '\\/')))
    }
    assert.match(markup, /object-cover/)
    assert.match(markup, /duration-2000/)
    assert.match(markup, /custom-hero-slide/)
    assert.match(markup, /is-active/)
    assert.match(markup, /aria-hidden="true"/)
  })

  it('loops from the fourth image back to the first', () => {
    assert.equal(getNextHeroSlideIndex(0, 4), 1)
    assert.equal(getNextHeroSlideIndex(3, 4), 0)
  })
})
