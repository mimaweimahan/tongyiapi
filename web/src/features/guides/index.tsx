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

import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { fetchGuides, type GuideCard } from './api'

export function GuidesPage() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isRoot = (auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  const [items, setItems] = useState<GuideCard[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    void (async () => {
      try {
        setLoading(true)
        setItems(await fetchGuides())
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Failed to load guides')
        )
      } finally {
        setLoading(false)
      }
    })()
  }, [t])

  return (
    <PublicLayout>
      <div className='mx-auto max-w-5xl px-4 py-10'>
        <p className='text-muted-foreground mb-2 text-sm'>
          <Link to='/' className='hover:underline'>
            {t('Home')}
          </Link>
          {' / '}
          {t('Guides')}
        </p>
        <div className='flex flex-wrap items-end justify-between gap-3'>
          <div>
            <h1 className='text-3xl font-bold tracking-tight'>{t('Guides')}</h1>
            <p className='text-muted-foreground mt-3 max-w-2xl text-sm'>
              {t(
                'Bring your own official keys. One site token, unified /v1 — billing stays on the vendor.'
              )}
            </p>
          </div>
          {isRoot && (
            <Button variant='outline' render={<Link to='/guides-admin' />}>
              {t('Edit guides')}
            </Button>
          )}
        </div>
        <div className='mt-8 grid gap-4 sm:grid-cols-2'>
          {items.map((item) => (
            <Link
              key={item.slug}
              to='/guides/$slug'
              params={{ slug: item.slug }}
              className='hover:bg-muted/40 block rounded-xl border p-5 transition-colors'
            >
              <h2 className='text-lg font-semibold'>{item.title}</h2>
              <p className='text-muted-foreground mt-2 text-sm leading-relaxed'>
                {item.description}
              </p>
            </Link>
          ))}
          {!loading && items.length === 0 && (
            <p className='text-muted-foreground'>{t('No guides yet.')}</p>
          )}
        </div>
      </div>
    </PublicLayout>
  )
}
