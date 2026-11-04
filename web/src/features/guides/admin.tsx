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
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { fetchGuide, fetchGuides, saveGuide, type GuideCard } from './api'

export function GuidesAdminPage() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isRoot = (auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  const [items, setItems] = useState<GuideCard[]>([])
  const [slug, setSlug] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [body, setBody] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!isRoot) return
    void (async () => {
      try {
        const list = await fetchGuides()
        setItems(list)
        if (list[0]) {
          setSlug(list[0].slug)
        }
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Failed to load guides')
        )
      }
    })()
  }, [isRoot, t])

  useEffect(() => {
    if (!slug || !isRoot) return
    void (async () => {
      try {
        const detail = await fetchGuide(slug)
        setTitle(detail.title)
        setDescription(detail.description)
        setBody(detail.body)
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : t('Failed to load guide')
        )
      }
    })()
  }, [slug, isRoot, t])

  if (!isRoot) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Edit guides')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <p className='text-muted-foreground text-sm'>
            {t('Only root can edit guides.')}
          </p>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const onSave = async () => {
    try {
      setSaving(true)
      await saveGuide(slug, { title, description, body })
      toast.success(t('Guide saved'))
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to save guide')
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Edit guides')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className='text-muted-foreground mb-4 text-sm'>
          {t('Update title, description and markdown. Slug stays the same.')}
        </p>
        <div className='mb-4'>
          <label className='mb-1 block text-sm font-medium'>{t('Guide')}</label>
          <select
            className='border-input bg-background h-9 w-full rounded-md border px-3 text-sm'
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
          >
            {items.map((item) => (
              <option key={item.slug} value={item.slug}>
                {item.slug} — {item.title}
              </option>
            ))}
          </select>
        </div>
        <div className='mb-4'>
          <label className='mb-1 block text-sm font-medium'>{t('Title')}</label>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} />
        </div>
        <div className='mb-4'>
          <label className='mb-1 block text-sm font-medium'>
            {t('Description')}
          </label>
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div className='mb-4'>
          <label className='mb-1 block text-sm font-medium'>
            {t('Body (Markdown)')}
          </label>
          <Textarea
            className='min-h-80 font-mono text-sm'
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </div>
        <div className='flex flex-wrap gap-3'>
          <Button disabled={saving || !slug} onClick={onSave}>
            {saving ? t('Saving...') : t('Save')}
          </Button>
          <Button variant='outline' render={<Link to='/guides' />}>
            {t('Public page')}
          </Button>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
