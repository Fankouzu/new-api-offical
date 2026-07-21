import { Suspense } from 'react'
import defaultSkin from '@/skins/default/manifest'
import { activeSkin } from './active-skin.gen'
import type { PublicPageKey } from './contracts'
import { SkinBoundary } from './skin-boundary'
import { SkinPageLoading } from './skin-page-loading'
import { createSkinPageRenderPlan } from './skin-page-plan'

type SkinPageProps = {
  page: PublicPageKey
}

export function SkinPage(props: SkinPageProps) {
  const plan = createSkinPageRenderPlan(activeSkin, defaultSkin, props.page)
  const Page = plan.definition.component
  const content = (
    <Suspense fallback={<SkinPageLoading />}>
      <Page />
    </Suspense>
  )

  if (plan.shell === 'self') {
    return content
  }

  const Shell = plan.Shell

  return (
    <SkinBoundary skinId={plan.skinId}>
      <Shell>{content}</Shell>
    </SkinBoundary>
  )
}
