import { createFileRoute } from '@tanstack/react-router'
import { PublicTopicPage } from '@/features/public-topic'

export const Route = createFileRoute('/providers/qwen-api')({
  component: () => <PublicTopicPage path='/providers/qwen-api' />,
})
