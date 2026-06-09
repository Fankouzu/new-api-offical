import { createFileRoute } from '@tanstack/react-router'
import { PublicTopicPage } from '@/features/public-topic'

export const Route = createFileRoute('/compare/ai-api-pricing')({
  component: () => <PublicTopicPage path='/compare/ai-api-pricing' />,
})
