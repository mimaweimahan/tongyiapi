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
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'

import { fetchGuide, type GuideDetail } from './api'

export function GuideDetailPage({ slug }: { slug: string }) {
  const { t } = useTranslation()
  const [item, setItem] = useState<GuideDetail | null>(null)

  useEffect(() => {
    void (async () => {
      try {
        setItem(await fetchGuide(slug))
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Failed to load guide')
        )
      }
    })()
  }, [slug, t])

  return (
    <PublicLayout>
      <div className='mx-auto max-w-3xl px-4 py-10'>
        <p className='text-muted-foreground mb-2 text-sm'>
          <Link to='/' className='hover:underline'>
            {t('Home')}
          </Link>
          {' / '}
          <Link to='/guides' className='hover:underline'>
            {t('Guides')}
          </Link>
          {' / '}
          {item?.title || slug}
        </p>
        <h1 className='text-3xl font-bold tracking-tight'>
          {item?.title || t('Loading...')}
        </h1>
        {item?.description && (
          <p className='text-muted-foreground mt-3 text-sm'>{item.description}</p>
        )}
        {item?.body && (
          <div className='mt-8'>
            <RichContent mode='markdown' content={item.body} />
          </div>
        )}
        <div className='mt-10'>
          <Button variant='outline' render={<Link to='/guides' />}>
            {t('All guides')}
          </Button>
        </div>
      </div>
    </PublicLayout>
  )
}
