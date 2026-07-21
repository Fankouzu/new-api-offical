import { Suspense } from 'react'
import { SkinBoundary } from './skin-boundary'
import { SkinPageLoading } from './skin-page-loading'
import type { SkinPageRenderPlan } from './skin-page-plan'

type SkinPageRendererProps = {
  plan: SkinPageRenderPlan
}

export function SkinPageRenderer(props: SkinPageRendererProps) {
  const Page = props.plan.definition.component
  const content = (
    <Suspense fallback={<SkinPageLoading />}>
      <Page />
    </Suspense>
  )

  if (props.plan.shell === 'self') {
    return content
  }

  const Shell = props.plan.Shell

  return (
    <SkinBoundary skinId={props.plan.skinId}>
      <Shell>{content}</Shell>
    </SkinBoundary>
  )
}
