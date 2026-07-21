import { createElement } from 'react'
import { LoadingState } from '@/components/loading-state'

export function SkinPageLoading() {
  return createElement(LoadingState, {
    className: 'min-h-[60vh]',
    size: 'lg',
  })
}
