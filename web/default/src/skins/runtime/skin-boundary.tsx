import {
  createElement,
  Fragment,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { SkinPortalProvider } from './skin-portal'

type SkinSurfaceDataset = {
  skinSurface?: string
}

export function setSkinSurface(
  dataset: SkinSurfaceDataset,
  skinId: string
): void {
  dataset.skinSurface = skinId
}

export function clearOwnedSkinSurface(
  dataset: SkinSurfaceDataset,
  skinId: string
): void {
  if (dataset.skinSurface === skinId) {
    delete dataset.skinSurface
  }
}

type SkinBoundaryProps = {
  skinId: string
  children: ReactNode
}

export function SkinBoundary(props: SkinBoundaryProps) {
  const [portalContainer, setPortalContainer] = useState<HTMLDivElement | null>(
    null
  )

  useEffect(() => {
    setSkinSurface(document.body.dataset, props.skinId)

    return () => {
      clearOwnedSkinSurface(document.body.dataset, props.skinId)
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
