import { activeSkin } from './active-skin.gen'
import { SkinPageRenderer } from './skin-page-renderer'
import { createSkinRouteRenderPlan } from './skin-route-plan'

type ActiveSkinRouteProps = {
  routeId: string
}

export function ActiveSkinRoute(props: ActiveSkinRouteProps) {
  const plan = createSkinRouteRenderPlan(activeSkin, props.routeId)

  return <SkinPageRenderer plan={plan} />
}
