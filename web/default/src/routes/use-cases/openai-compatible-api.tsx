import { createFileRoute } from '@tanstack/react-router'
import { PublicTopicPage } from '@/features/public-topic'

export const Route = createFileRoute('/use-cases/openai-compatible-api')({
  component: () => <PublicTopicPage path='/use-cases/openai-compatible-api' />,
})
