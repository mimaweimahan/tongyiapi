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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { fetchByokPrices, syncByokPrices, type ByokVendorPrice } from './api'

export function ByokPricesPage() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isRoot = (auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  const [items, setItems] = useState<ByokVendorPrice[]>([])
  const [vendors, setVendors] = useState<Record<string, string>>({})
  const [vendor, setVendor] = useState('')
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)

  const refresh = async () => {
    try {
      setLoading(true)
      const data = await fetchByokPrices()
      setItems(data.items || [])
      setVendors(data.vendors || {})
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to load prices')
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return items.filter((row) => {
      if (vendor && row.provider !== vendor) return false
      if (!q) return true
      return (
        row.model.toLowerCase().includes(q) ||
        row.provider.toLowerCase().includes(q) ||
        (vendors[row.provider] || '').toLowerCase().includes(q)
      )
    })
  }, [items, query, vendor, vendors])

  const onSync = async () => {
    try {
      setSyncing(true)
      await syncByokPrices('all')
      toast.success(t('Price table refreshed'))
      await refresh()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to sync prices')
      )
    } finally {
      setSyncing(false)
    }
  }

  const unitLabel = (row: ByokVendorPrice) => {
    if (row.unit === 'token_m') {
      return row.currency === 'USD'
        ? t('USD / million tokens')
        : t('CNY / million tokens')
    }
    return row.currency === 'USD' ? t('USD / call') : t('CNY / call')
  }

  return (
    <PublicLayout>
      <div className='mx-auto max-w-6xl px-4 py-10'>
        <p className='text-muted-foreground mb-2 text-sm'>
          <Link to='/' className='hover:underline'>
            {t('Home')}
          </Link>
          {' / '}
          {t('Official Pricing')}
        </p>
        <h1 className='text-3xl font-bold tracking-tight'>
          {t('Official Pricing')}
        </h1>
        <p className='text-muted-foreground mt-3 max-w-3xl text-sm leading-relaxed'>
          {t(
            'SiliconFlow is priced in CNY; Gemini uses official USD rates. This page mirrors vendor list prices for your own keys — not this site wallet quota.'
          )}
        </p>
        <div className='mt-6 flex flex-wrap items-center gap-2'>
          <Button
            variant={vendor === '' ? 'default' : 'outline'}
            size='sm'
            onClick={() => setVendor('')}
          >
            {t('All')}
          </Button>
          {Object.entries(vendors).map(([id, name]) => (
            <Button
              key={id}
              variant={vendor === id ? 'default' : 'outline'}
              size='sm'
              onClick={() => setVendor(id)}
            >
              {name}
            </Button>
          ))}
          <Input
            className='ml-auto max-w-xs'
            placeholder={t('Filter models')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {isRoot && (
            <Button size='sm' variant='outline' disabled={syncing} onClick={onSync}>
              {syncing ? t('Syncing...') : t('Sync prices')}
            </Button>
          )}
        </div>
        <p className='text-muted-foreground mt-3 text-sm'>
          {loading
            ? t('Loading...')
            : t('Showing {{count}} models', { count: filtered.length })}
        </p>
        <div className='mt-4 overflow-auto rounded-lg border'>
          <table className='w-full min-w-[640px] text-sm'>
            <thead className='bg-muted/40 text-left'>
              <tr>
                <th className='px-3 py-2 font-medium'>{t('Vendor')}</th>
                <th className='px-3 py-2 font-medium'>{t('Model')}</th>
                <th className='px-3 py-2 text-right font-medium'>{t('Input')}</th>
                <th className='px-3 py-2 text-right font-medium'>{t('Output')}</th>
                <th className='px-3 py-2 font-medium'>{t('Unit')}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((row) => (
                <tr key={`${row.provider}:${row.model}`} className='border-t'>
                  <td className='px-3 py-2'>
                    {vendors[row.provider] || row.provider}
                  </td>
                  <td className='px-3 py-2 font-mono text-xs'>{row.model}</td>
                  <td className='px-3 py-2 text-right tabular-nums'>
                    {row.input_cny}
                  </td>
                  <td className='px-3 py-2 text-right tabular-nums'>
                    {row.output_cny}
                  </td>
                  <td className='text-muted-foreground px-3 py-2'>
                    {unitLabel(row)}
                  </td>
                </tr>
              ))}
              {!loading && filtered.length === 0 && (
                <tr>
                  <td
                    colSpan={5}
                    className='text-muted-foreground px-3 py-8 text-center'
                  >
                    {t('No price rows yet. Root can sync from official sources.')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className='mt-6 flex flex-wrap gap-3'>
          <Button render={<Link to='/usage-cost' />}>
            {t('Estimate my usage')}
          </Button>
          <Button variant='outline' render={<Link to='/guides' />}>
            {t('Guides')}
          </Button>
        </div>
      </div>
    </PublicLayout>
  )
}
