/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { removeTrailingSlash } from './utils'

export interface BinancePaySettingsValues {
  BinancePayEnabled: boolean
  BinancePaySandbox: boolean
  BinancePayApiKey: string
  BinancePayApiSecret: string
  BinancePayWebhookPubKey: string
  BinancePayReturnURL: string
  BinancePayCurrency: string
  BinancePayUnitPrice: number
  BinancePayMinTopUp: number
}

interface Props {
  defaultValues: BinancePaySettingsValues
}

export function BinancePaySettingsSection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [loading, setLoading] = useState(false)
  const form = useForm<BinancePaySettingsValues>({
    defaultValues: props.defaultValues,
  })

  useEffect(() => {
    form.reset(props.defaultValues)
  }, [props.defaultValues, form])

  const handleSave = async () => {
    const values = form.getValues()
    const enabled = !!values.BinancePayEnabled

    if (enabled && !values.BinancePayApiKey.trim()) {
      toast.error(t('Binance Pay API key is required'))
      return
    }
    if (enabled && !values.BinancePayApiSecret.trim()) {
      toast.error(t('Binance Pay API secret is required'))
      return
    }
    if (enabled && !values.BinancePayWebhookPubKey.trim()) {
      toast.error(t('Binance Pay webhook public key is required'))
      return
    }
    if (enabled && Number(values.BinancePayUnitPrice) <= 0) {
      toast.error(t('Unit price must be greater than 0'))
      return
    }
    if (enabled && Number(values.BinancePayMinTopUp) < 1) {
      toast.error(t('Minimum top-up amount must be at least 1'))
      return
    }

    setLoading(true)
    try {
      const options: { key: string; value: string }[] = [
        { key: 'BinancePayEnabled', value: enabled ? 'true' : 'false' },
        {
          key: 'BinancePaySandbox',
          value: values.BinancePaySandbox ? 'true' : 'false',
        },
        {
          key: 'BinancePayReturnURL',
          value: removeTrailingSlash(values.BinancePayReturnURL || ''),
        },
        {
          key: 'BinancePayCurrency',
          value: values.BinancePayCurrency || 'USDT',
        },
        {
          key: 'BinancePayUnitPrice',
          value: String(values.BinancePayUnitPrice ?? 1),
        },
        {
          key: 'BinancePayMinTopUp',
          value: String(values.BinancePayMinTopUp ?? 1),
        },
      ]

      if ((values.BinancePayApiKey || '').trim()) {
        options.push({ key: 'BinancePayApiKey', value: values.BinancePayApiKey })
      }
      if ((values.BinancePayApiSecret || '').trim()) {
        options.push({
          key: 'BinancePayApiSecret',
          value: values.BinancePayApiSecret,
        })
      }
      if ((values.BinancePayWebhookPubKey || '').trim()) {
        options.push({
          key: 'BinancePayWebhookPubKey',
          value: values.BinancePayWebhookPubKey,
        })
      }

      for (const option of options) {
        await updateOption.mutateAsync(option)
      }
      toast.success(t('Updated successfully'))
    } catch {
      toast.error(t('Update failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <SettingsSection
      title={t('Binance Pay Payment Gateway')}
      description={t('Configure Binance Pay merchant checkout for balance top-ups')}
    >
      <Alert>
        <AlertDescription className='text-xs'>
          {t(
            'Configure the webhook URL in Binance Pay: <ServerAddress>/api/binance-pay/webhook'
          )}
        </AlertDescription>
      </Alert>

      <div className='grid grid-cols-3 gap-4'>
        <div className='flex items-center gap-2'>
          <Switch
            checked={form.watch('BinancePayEnabled')}
            onCheckedChange={(value) => form.setValue('BinancePayEnabled', value)}
          />
          <Label>{t('Enable Binance Pay')}</Label>
        </div>
        <div className='flex items-center gap-2'>
          <Switch
            checked={form.watch('BinancePaySandbox')}
            onCheckedChange={(value) => form.setValue('BinancePaySandbox', value)}
          />
          <Label>{t('Sandbox mode')}</Label>
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Currency')}</Label>
          <Input placeholder='USDT' {...form.register('BinancePayCurrency')} />
        </div>
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('API key')}</Label>
          <Input
            placeholder={t('Leave blank to keep the existing key')}
            {...form.register('BinancePayApiKey')}
          />
          <p className='text-muted-foreground text-xs'>
            {t('Stored value is not echoed back for security')}
          </p>
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('API secret')}</Label>
          <Input
            type='password'
            placeholder={t('Leave blank to keep the existing secret')}
            {...form.register('BinancePayApiSecret')}
          />
        </div>
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Webhook public key')}</Label>
          <Textarea
            rows={4}
            placeholder={t('Leave blank to keep the existing key')}
            {...form.register('BinancePayWebhookPubKey')}
            className='font-mono text-xs'
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Payment return URL')}</Label>
          <Input
            placeholder='https://example.com/console/topup'
            {...form.register('BinancePayReturnURL')}
          />
          <p className='text-muted-foreground text-xs'>
            {t('Defaults to the wallet page when empty')}
          </p>
        </div>
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Unit price (crypto amount / USD)')}</Label>
          <Input
            type='number'
            step={0.01}
            min={0}
            {...form.register('BinancePayUnitPrice', { valueAsNumber: true })}
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Minimum top-up (USD)')}</Label>
          <Input
            type='number'
            min={1}
            {...form.register('BinancePayMinTopUp', { valueAsNumber: true })}
          />
        </div>
      </div>

      <Button onClick={handleSave} disabled={loading}>
        {loading ? t('Saving...') : t('Save Binance Pay settings')}
      </Button>
    </SettingsSection>
  )
}
