import { createFileRoute } from '@tanstack/react-router'
import { PublicTopicPage } from '@/features/public-topic'

export const Route = createFileRoute('/providers/gemini-api')({
  component: () => <PublicTopicPage path='/providers/gemini-api' />,
})
