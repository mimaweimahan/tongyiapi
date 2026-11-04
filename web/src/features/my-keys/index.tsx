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
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { getMyKeys } from './api'
import { ProviderForm } from './components/provider-form'
import { SiteTokenForm } from './components/site-token-form'
import type { ByokProvider, MyKeyLink, MyKeysState } from './types'

export function MyKeys() {
  const { t } = useTranslation()
  const [state, setState] = useState<MyKeysState | null>(null)
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<ByokProvider | null>(null)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getMyKeys()
      setState(data)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to load my keys')
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void refresh()
  }, [refresh])

  useEffect(() => {
    if (typeof window === 'undefined') return
    if (window.location.hash !== '#token') return
    const timer = window.setTimeout(() => {
      document
        .getElementById('token')
        ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 80)
    return () => window.clearTimeout(timer)
  }, [state])

  const links: Record<string, MyKeyLink> = state?.links || {}
  const connectedCount = Object.keys(links).length
  const total = state?.providers?.length || 0
  const selectedLink = selected ? links[selected.id] : undefined

  function clearSelected() {
    setSelected(null)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('My Keys')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className='mb-4 text-sm text-muted-foreground'>
          {t(
            'Upload your own official vendor keys. Billing stays on the vendor side; this site only routes. Paid vendors need activation first — after that you still use your own official API key; this site does not resell shared keys.'
          )}
        </p>
        <div className='mb-4 flex flex-wrap items-center gap-3 text-sm text-muted-foreground'>
          <span>
            {t('Connected {{connected}} / {{total}}', {
              connected: connectedCount,
              total,
            })}
          </span>
          {state?.group ? (
            <Badge variant='secondary'>
              {t('Group')}: {state.group}
            </Badge>
          ) : null}
          <Button variant='outline' size='sm' onClick={() => void refresh()}>
            {t('Refresh')}
          </Button>
          <Button
            variant='ghost'
            size='sm'
            onClick={() =>
              document
                .getElementById('token')
                ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
            }
          >
            {t('Create site token')}
          </Button>
        </div>

        {loading && !state ? (
          <p className='text-sm text-muted-foreground'>{t('Loading…')}</p>
        ) : (
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
            {(state?.providers || []).map((provider) => {
              const link = links[provider.id]
              const connected = Boolean(link)
              const locked = provider.can_connect === false
              return (
                <button
                  key={provider.id}
                  type='button'
                  onClick={() => setSelected(provider)}
                  className='rounded-xl border border-border bg-card p-4 text-left transition hover:border-foreground/30 hover:bg-accent/40'
                >
                  <div className='mb-2 flex items-start justify-between gap-2'>
                    <div className='flex items-center gap-2 font-medium'>
                      <img
                        src={(provider.logo_url || '').trim() || `/logos/${provider.id}.svg`}
                        alt=''
                        className='h-5 w-5 shrink-0 object-contain'
                      />
                      <span>{provider.name}</span>
                    </div>
                    <div className='flex flex-wrap justify-end gap-1'>
                      {provider.pricing_tier === 'paid' ? (
                        <Badge variant='secondary'>{t('Paid')}</Badge>
                      ) : null}
                      <Badge
                        variant={
                          locked
                            ? 'destructive'
                            : connected
                              ? 'default'
                              : 'outline'
                        }
                      >
                        {locked
                          ? t('Requires activation')
                          : connected
                            ? t('Connected')
                            : t('Connect')}
                      </Badge>
                    </div>
                  </div>
                  <p className='text-xs text-muted-foreground'>
                    {locked
                      ? t('Ask admin to activate; then use your own official key')
                      : connected
                        ? link?.key_hint || '••••'
                        : t('Type {{type}}', { type: provider.type })}
                  </p>
                </button>
              )
            })}
          </div>
        )}

        <Dialog
          open={selected !== null}
          onOpenChange={(open) => {
            if (!open) clearSelected()
          }}
        >
          <DialogContent className='sm:max-w-lg'>
            {selected ? (
              <>
                <DialogHeader>
                  <DialogTitle className='flex items-center gap-2 pr-8'>
                    <img
                      src={(selected.logo_url || '').trim() || `/logos/${selected.id}.svg`}
                      alt=''
                      className='h-6 w-6 shrink-0 object-contain'
                    />
                    <span>{selected.name}</span>
                  </DialogTitle>
                  <DialogDescription className='flex flex-wrap items-center gap-x-3 gap-y-1'>
                    <span>
                      {selected.can_connect === false
                        ? t('Requires activation')
                        : selectedLink
                          ? t('Connected · {{hint}}', {
                              hint: selectedLink.key_hint || '••••',
                            })
                          : t('Not connected yet')}
                    </span>
                    {selected.can_connect !== false && selected.key_url ? (
                      <a
                        href={selected.key_url}
                        target='_blank'
                        rel='noopener noreferrer'
                        className='text-sm font-medium text-primary underline-offset-4 hover:underline'
                      >
                        {t('Get API Key')}
                      </a>
                    ) : null}
                  </DialogDescription>
                </DialogHeader>
                {selected.can_connect === false ? (
                  <div className='space-y-3 text-sm'>
                    <p>
                      {t(
                        'This paid vendor is locked until an admin activates it for your account.'
                      )}
                    </p>
                    <p className='text-muted-foreground'>
                      {t(
                        'After activation you still use your own official API key. This site does not hold or resell shared keys.'
                      )}
                    </p>
                    <div className='flex justify-end'>
                      <Button variant='outline' onClick={clearSelected}>
                        {t('Close')}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <ProviderForm
                    key={selected.id}
                    provider={selected}
                    link={selectedLink}
                    embedded
                    onCancel={clearSelected}
                    onDone={() => {
                      clearSelected()
                      void refresh()
                    }}
                  />
                )}
              </>
            ) : null}
          </DialogContent>
        </Dialog>

        <SiteTokenForm links={links} group={state?.group} />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
