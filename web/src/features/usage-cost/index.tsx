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
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

type UsageItem = {
  created_at: number
  model_name: string
  prompt_tokens: number
  completion_tokens: number
  currency?: string
  cost?: string
  priced: boolean
}

type UsageData = {
  items: UsageItem[]
  totals: Record<string, string>
  known: number
  count: number
}

export function UsageCostPage() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const [data, setData] = useState<UsageData | null>(null)

  useEffect(() => {
    if (!auth.user) return
    void (async () => {
      try {
        const response = await api.get<{
          success: boolean
          message?: string
          data: UsageData
        }>('/api/byok/usage-cost', {
          skipBusinessError: true,
          skipErrorHandler: true,
          disableDuplicate: true,
        })
        setData(requireServerSuccess(response.data).data)
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Failed to load usage cost')
        )
      }
    })()
  }, [auth.user, t])

  if (!auth.user) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Usage estimate')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <p className='text-muted-foreground mb-4 text-sm'>
            {t('Sign in to estimate official vendor cost from your logs.')}
          </p>
          <Button render={<Link to='/sign-in' />}>{t('Sign in')}</Button>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const totalsText = data
    ? Object.entries(data.totals || {})
        .map(([currency, amount]) =>
          currency === 'USD' ? `$ ${amount}` : `¥ ${amount}`
        )
        .join(' / ') || '—'
    : '—'

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Usage estimate')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className='text-muted-foreground mb-4 text-sm'>
          {t(
            'Token counts come from your logs. SiliconFlow estimated in CNY, Gemini in USD. Currencies are not converted. Not this site wallet quota.'
          )}
        </p>
        <p className='mb-2 text-2xl font-semibold tabular-nums'>{totalsText}</p>
        <p className='text-muted-foreground mb-6 text-sm'>
          {data
            ? t('{{count}} calls, {{known}} matched price rows', {
                count: data.count,
                known: data.known,
              })
            : t('Loading...')}
        </p>
        <div className='overflow-auto rounded-lg border'>
          <table className='w-full min-w-[640px] text-sm'>
            <thead className='bg-muted/40 text-left'>
              <tr>
                <th className='px-3 py-2'>{t('Time')}</th>
                <th className='px-3 py-2'>{t('Model')}</th>
                <th className='px-3 py-2 text-right'>{t('Input')}</th>
                <th className='px-3 py-2 text-right'>{t('Output')}</th>
                <th className='px-3 py-2 text-right'>{t('Estimate')}</th>
              </tr>
            </thead>
            <tbody>
              {(data?.items || []).map((row, index) => (
                <tr key={`${row.created_at}-${index}`} className='border-t'>
                  <td className='px-3 py-2'>
                    {row.created_at
                      ? new Date(row.created_at * 1000).toLocaleString()
                      : ''}
                  </td>
                  <td className='px-3 py-2 font-mono text-xs'>{row.model_name}</td>
                  <td className='px-3 py-2 text-right tabular-nums'>
                    {row.prompt_tokens}
                  </td>
                  <td className='px-3 py-2 text-right tabular-nums'>
                    {row.completion_tokens}
                  </td>
                  <td className='px-3 py-2 text-right tabular-nums'>
                    {row.priced
                      ? `${row.currency === 'USD' ? '$' : '¥'} ${row.cost}`
                      : t('No price')}
                  </td>
                </tr>
              ))}
              {data && data.items.length === 0 && (
                <tr>
                  <td
                    colSpan={5}
                    className='text-muted-foreground px-3 py-8 text-center'
                  >
                    {t('No consume logs yet.')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className='mt-6'>
          <Button variant='outline' render={<Link to='/prices' />}>
            {t('Official Pricing')}
          </Button>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
