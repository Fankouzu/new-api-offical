import { createFileRoute } from '@tanstack/react-router'
import { PublicTopicPage } from '@/features/public-topic'

export const Route = createFileRoute('/providers/deepseek-api')({
  component: () => <PublicTopicPage path='/providers/deepseek-api' />,
})
