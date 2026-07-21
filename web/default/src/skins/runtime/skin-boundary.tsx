import {
  createElement,
  Fragment,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { SkinBoundaryPresenceContext } from './skin-boundary-context'
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
  const hasAncestorBoundary = useContext(SkinBoundaryPresenceContext)
  const [surfaceToken] = useState(createSkinSurfaceToken)
  const [portalContainer, setPortalContainer] = useState<HTMLDivElement | null>(
    null
  )

  if (hasAncestorBoundary) {
    throw new Error('SkinBoundary cannot be nested')
  }

  useEffect(() => {
    const dataset = document.body.dataset
    registerSkinSurface(dataset, surfaceToken, props.skinId)

    return () => {
      unregisterSkinSurface(dataset, surfaceToken)
    }
  }, [props.skinId, surfaceToken])

  return createElement(
    SkinBoundaryPresenceContext.Provider,
    { value: true },
    createElement(
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
  )
}
