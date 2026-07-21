import {
  createElement,
  Fragment,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { SkinPortalProvider } from './skin-portal'
import {
  createSkinSurfaceToken,
  registerSkinSurface,
  unregisterSkinSurface,
} from './skin-surface-registry'

type SkinBoundaryProps = {
  skinId: string
  children: ReactNode
}

export function SkinBoundary(props: SkinBoundaryProps) {
  const surfaceToken = useRef(createSkinSurfaceToken())
  const [portalContainer, setPortalContainer] = useState<HTMLDivElement | null>(
    null
  )

  useEffect(() => {
    const dataset = document.body.dataset
    const token = surfaceToken.current
    registerSkinSurface(dataset, token, props.skinId)

    return () => {
      unregisterSkinSurface(dataset, token)
    }
  }, [props.skinId])

  return createElement(
    'div',
    { 'data-skin': props.skinId, 'data-skin-boundary': true },
    createElement(SkinPortalProvider, {
      container: portalContainer,
      children: createElement(
        Fragment,
        null,
        props.children,
        createElement('div', {
          ref: setPortalContainer,
          'data-skin-portal-root': props.skinId,
        })
      ),
    })
  )
}
