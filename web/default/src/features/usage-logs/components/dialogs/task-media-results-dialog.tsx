import { Copy, ExternalLink, ImageIcon, Video } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatusBadge } from '@/components/status-badge'
import type { TaskMediaResult } from '../../lib/task-media-results'

interface TaskMediaResultsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  results: TaskMediaResult[]
  taskId?: string
}

function copyUrl(url: string, successMessage: string): void {
  void navigator.clipboard.writeText(url).then(() => {
    toast.success(successMessage)
  })
}

function openExternalUrl(url: string): void {
  window.open(url, '_blank', 'noopener,noreferrer')
}

function TaskMediaCard({ result }: { result: TaskMediaResult }) {
  const { t } = useTranslation()
  const isImage = result.type === 'image'
  const title = isImage ? t('Generated image') : t('Generated video')

  return (
    <div className='bg-card overflow-hidden rounded-lg border'>
      <div className='bg-muted/40 flex min-h-[220px] items-center justify-center'>
        {isImage ? (
          <img
            src={result.url}
            alt={title}
            loading='lazy'
            className='max-h-[420px] w-full object-contain'
          />
        ) : (
          <video
            src={result.url}
            controls
            preload='metadata'
            className='max-h-[420px] w-full object-contain'
          />
        )}
      </div>
      <div className='space-y-3 p-3'>
        <div className='flex items-center justify-between gap-2'>
          <StatusBadge
            label={isImage ? t('Image') : t('Video')}
            variant={isImage ? 'blue' : 'purple'}
            copyable={false}
            showDot={false}
          />
          <div className='flex shrink-0 items-center gap-1'>
            <Button
              variant='ghost'
              size='sm'
              className='h-7 gap-1 px-2 text-xs'
              onClick={() => copyUrl(result.url, t('Copied'))}
            >
              <Copy className='size-3' />
              {t('Copy Link')}
            </Button>
            <Button
              variant='ghost'
              size='sm'
              className='h-7 gap-1 px-2 text-xs'
              onClick={() => openExternalUrl(result.url)}
            >
              <ExternalLink className='size-3' />
              {t('Open')}
            </Button>
          </div>
        </div>
        <p className='text-muted-foreground font-mono text-xs break-all'>
          {result.url}
        </p>
      </div>
    </div>
  )
}

export function TaskMediaResultsDialog(props: TaskMediaResultsDialogProps) {
  const { t } = useTranslation()
  const results = Array.isArray(props.results) ? props.results : []

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-5xl'>
        <DialogHeader>
          <DialogTitle>{t('Generated Results')}</DialogTitle>
          <DialogDescription>
            {props.taskId
              ? `${t('Task ID:')} ${props.taskId}`
              : t('View generated media results')}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[75vh] pr-4'>
          <div className='grid gap-4 py-4 md:grid-cols-2'>
            {results.map((result) => (
              <TaskMediaCard key={`${result.type}:${result.url}`} result={result} />
            ))}
          </div>
          {results.length === 0 && (
            <div className='text-muted-foreground flex flex-col items-center justify-center gap-2 py-12 text-sm'>
              <div className='flex items-center gap-2'>
                <ImageIcon className='size-4' />
                <Video className='size-4' />
              </div>
              {t('No generated media results')}
            </div>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
