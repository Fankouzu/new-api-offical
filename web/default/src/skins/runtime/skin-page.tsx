import defaultSkin from '@/skins/default/manifest'
import { activeSkin } from './active-skin.gen'
import type { PublicPageKey } from './contracts'
import { createSkinPageRenderPlan } from './skin-page-plan'
import { SkinPageRenderer } from './skin-page-renderer'

type SkinPageProps = {
  page: PublicPageKey
}

export function SkinPage(props: SkinPageProps) {
  const plan = createSkinPageRenderPlan(activeSkin, defaultSkin, props.page)
  return <SkinPageRenderer plan={plan} />
}
