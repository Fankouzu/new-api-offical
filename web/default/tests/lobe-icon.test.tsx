import { describe, expect, test } from 'bun:test'
import { isValidElement, type ReactElement, type ReactNode } from 'react'

import { getLobeIcon } from '../src/lib/lobe-icon'

interface ImageIconProps {
  alt?: string
  src?: string
}

function getImageProps(node: ReactNode): ImageIconProps {
  if (!isValidElement<ImageIconProps>(node)) {
    throw new Error('Expected a valid React element')
  }

  const element: ReactElement<ImageIconProps> = node
  return element.props
}

describe('getLobeIcon', () => {
  test('renders external logo URLs even when channel callers append color suffix', () => {
    const props = getImageProps(getLobeIcon('https://kie.ai/logo.png.Color', 16))

    expect(props.src).toBe('https://kie.ai/logo.png')
    expect(props.alt).toBe('')
  })
})
