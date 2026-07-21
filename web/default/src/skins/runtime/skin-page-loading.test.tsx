import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { LoadingState } from '@/components/loading-state'
import { SkinPageLoading } from './skin-page-loading'

describe('SkinPageLoading', () => {
  it('renders an explicit loading state with stable public-page height', () => {
    const loading = SkinPageLoading()

    assert.equal(loading.type, LoadingState)
    assert.equal(loading.props.className, 'min-h-[60vh]')
    assert.equal(loading.props.size, 'lg')
  })
})
