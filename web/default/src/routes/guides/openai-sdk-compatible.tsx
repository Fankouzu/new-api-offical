import { createFileRoute } from '@tanstack/react-router'
import { PublicTopicPage } from '@/features/public-topic'

export const Route = createFileRoute('/guides/openai-sdk-compatible')({
  component: () => <PublicTopicPage path='/guides/openai-sdk-compatible' />,
})
