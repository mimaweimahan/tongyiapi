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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'

import { createSiteToken } from '../api'
import type { MyKeyLink, SiteTokenResult } from '../types'

type SiteTokenFormProps = {
  links: Record<string, MyKeyLink>
  group?: string
}

export function SiteTokenForm({ links, group }: SiteTokenFormProps) {
  const { t } = useTranslation()
  const vendors = useMemo(
    () =>
      Object.values(links).sort((a, b) =>
        a.provider.localeCompare(b.provider)
      ),
    [links]
  )
  const [provider, setProvider] = useState('all')
  const [model, setModel] = useState('auto')
  const [name, setName] = useState('site')
  const [creating, setCreating] = useState(false)
  const [result, setResult] = useState<SiteTokenResult | null>(null)

  const chatModels = useMemo(() => {
    if (provider === 'all') return [] as string[]
    const link = links[provider]
    if (!link) return []
    if (link.chat_models && link.chat_models.length > 0) {
      return link.chat_models
    }
    return (link.models || '')
      .split(',')
      .map((item) => item.trim())
      .filter((item) => item && item !== 'auto' && item !== 'auto-chat')
  }, [links, provider])

  function handleProviderChange(next: string) {
    setProvider(next)
    setModel('auto')
    setResult(null)
  }

  async function handleCreate() {
    if (!vendors.length) {
      toast.error(t('Connect at least one vendor key first'))
      return
    }
    try {
      setCreating(true)
      const token = await createSiteToken({
        provider,
        model,
        name: name.trim() || 'site',
      })
      setResult(token)
      toast.success(t('Site token created'))
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to create site token')
      )
    } finally {
      setCreating(false)
    }
  }

  async function copyText(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value)
      toast.success(t('Copied {{label}}', { label }))
    } catch {
      toast.error(t('Copy failed'))
    }
  }

  if (!vendors.length) {
    return (
      <section
        id='token'
        className='mt-8 space-y-3 rounded-xl border border-border bg-card p-4'
      >
        <h2 className='text-base font-semibold'>{t('Create site token')}</h2>
        <p className='text-sm text-muted-foreground'>
          {t('Connect at least one vendor key above before creating a token.')}
        </p>
      </section>
    )
  }

  return (
    <section
      id='token'
      className='mt-8 space-y-4 rounded-xl border border-border bg-card p-4'
    >
      <div>
        <h2 className='text-base font-semibold'>{t('Create site token')}</h2>
        <p className='text-sm text-muted-foreground'>
          {t(
            'Pick a connected vendor or all vendors, then a model or auto. Vendor + auto locks that vendor chat model list. Group stays {{group}}.',
            { group: group || 'u{id}' }
          )}
        </p>
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='space-y-2'>
          <Label htmlFor='tok-vendor'>{t('Vendor')}</Label>
          <NativeSelect
            id='tok-vendor'
            className='w-full'
            value={provider}
            onChange={(event) => handleProviderChange(event.target.value)}
          >
            <NativeSelectOption value='all'>
              {t('All connected (recommend auto)')}
            </NativeSelectOption>
            {vendors.map((item) => (
              <NativeSelectOption key={item.provider} value={item.provider}>
                {item.provider} · {item.key_hint || '••••'}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='tok-model'>{t('Model')}</Label>
          <NativeSelect
            id='tok-model'
            className='w-full'
            value={model}
            onChange={(event) => {
              setModel(event.target.value)
              setResult(null)
            }}
          >
            <NativeSelectOption value='auto'>
              {t('auto (failover)')}
            </NativeSelectOption>
            {chatModels.map((item) => (
              <NativeSelectOption key={item} value={item}>
                {item}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
      </div>

      <div className='space-y-2'>
        <Label htmlFor='tok-name'>{t('Token name')}</Label>
        <Input
          id='tok-name'
          maxLength={40}
          placeholder={t('e.g. cursor or xinyuan')}
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
      </div>

      <Button onClick={handleCreate} disabled={creating}>
        {creating ? t('Creating…') : t('Generate token')}
      </Button>

      {result ? (
        <div className='space-y-3 rounded-lg border border-border bg-muted/30 p-3 text-sm'>
          <p className='font-medium'>
            {t('Token created. Copy it now — shown only once.')}
          </p>
          <div className='flex flex-wrap items-center gap-2'>
            <span className='text-muted-foreground'>{t('Base URL')}</span>
            <code className='break-all'>{result.base_url}</code>
            <Button
              size='sm'
              variant='outline'
              onClick={() => void copyText(result.base_url, t('Base URL'))}
            >
              {t('Copy')}
            </Button>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <span className='text-muted-foreground'>{t('API Key')}</span>
            <code className='break-all'>{result.key}</code>
            <Button
              size='sm'
              variant='outline'
              onClick={() => void copyText(result.key, t('API Key'))}
            >
              {t('Copy')}
            </Button>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <span className='text-muted-foreground'>{t('Model')}</span>
            <code>{result.model}</code>
            <Button
              size='sm'
              variant='outline'
              onClick={() => void copyText(result.model, t('Model'))}
            >
              {t('Copy')}
            </Button>
          </div>
          <div className='flex flex-wrap gap-4 text-muted-foreground'>
            <span>
              {t('Group')}: <code>{result.group}</code>
            </span>
            <span>
              {t('Scope')}:{' '}
              {result.scope === 'vendor'
                ? t('{{provider}} + auto (vendor only)', {
                    provider: result.provider,
                  })
                : t('All connected vendors')}
            </span>
          </div>
        </div>
      ) : null}
    </section>
  )
}
