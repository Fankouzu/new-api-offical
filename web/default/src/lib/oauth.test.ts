import assert from 'node:assert/strict'

import { buildLinuxDOOAuthUrl } from './oauth.ts'

const originalWindow = globalThis.window

Object.defineProperty(globalThis, 'window', {
  configurable: true,
  value: {
    location: {
      origin: 'https://lizh.ai',
    },
  },
})

try {
  const url = new URL(buildLinuxDOOAuthUrl('linux-client-id', 'state-token'))

  assert.equal(url.origin + url.pathname, 'https://connect.linux.do/oauth2/authorize')
  assert.equal(url.searchParams.get('response_type'), 'code')
  assert.equal(url.searchParams.get('client_id'), 'linux-client-id')
  assert.equal(url.searchParams.get('state'), 'state-token')
  assert.equal(url.searchParams.get('redirect_uri'), 'https://lizh.ai/oauth/linuxdo')
} finally {
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: originalWindow,
  })
}
