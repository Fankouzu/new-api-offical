import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  MODEL_HERO_SLIDES,
  getNextModelSlideIndex,
} from './custom-model-hero-data'

describe('CustomModelHero', () => {
  it('keeps the approved model and image order', () => {
    assert.deepEqual(
      MODEL_HERO_SLIDES.map((slide) => slide.name),
      ['Kimi K3', 'GLM 5.2', 'Qwen 3.8', 'DeepSeek V4', 'Mimo 2.5']
    )
    assert.equal(MODEL_HERO_SLIDES.length, 5)
  })

  it('loops from the fifth slide back to the first', () => {
    assert.equal(getNextModelSlideIndex(0, 5), 1)
    assert.equal(getNextModelSlideIndex(4, 5), 0)
  })
})
