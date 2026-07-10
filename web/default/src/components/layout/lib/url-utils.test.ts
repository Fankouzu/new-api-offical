import assert from 'node:assert/strict'
import { test } from 'node:test'
import { urlToString } from './url-utils'

test('urlToString does not duplicate query params when pathname already has search', () => {
  const href = urlToString({
    pathname: '/usage-logs/common?page=2',
    search: '?page=2',
  } as never)

  assert.equal(href, '/usage-logs/common?page=2')
})

test('urlToString prefixes object search with a single question mark', () => {
  const href = urlToString({
    pathname: '/usage-logs/common',
    search: 'page=2',
  } as never)

  assert.equal(href, '/usage-logs/common?page=2')
})
