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
import { ArrowDown, ArrowUp, Plus } from 'lucide-react'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  createByokProvider,
  fetchByokEntitlements,
  fetchByokProvidersAdmin,
  grantByokEntitlement,
  reorderByokProviders,
  revokeByokEntitlement,
  setByokProviderEnabled,
  updateByokProvider,
  type ByokEntitlement,
  type ByokProviderAdmin,
  type ByokProviderInput,
} from './api'

const emptyForm = (): ByokProviderInput => ({
  id: '',
  name: '',
  channel_type: 1,
  models: '',
  key_hint: 'sk-...',
  need_base: false,
  base_hint: '',
  other: '',
  key_url: '',
  logo_url: '',
  enabled: true,
  sort: 1000,
  is_openai_compatible: false,
  pricing_tier: 'free',
})

function providerLogoSrc(p: { id: string; logo_url?: string }) {
  const url = (p.logo_url || '').trim()
  return url || `/logos/${p.id}.svg`
}

export function ByokProvidersAdminPage() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isRoot = (auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  const [items, setItems] = useState<ByokProviderAdmin[]>([])
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<ByokProviderInput | null>(null)
  const [creating, setCreating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [entitlements, setEntitlements] = useState<ByokEntitlement[]>([])
  const [entUser, setEntUser] = useState('')
  const [entProvider, setEntProvider] = useState('')
  const [entNote, setEntNote] = useState('')
  const [entBusy, setEntBusy] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const list = await fetchByokProvidersAdmin()
      setItems(list)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to load BYOK providers')
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  const refreshEntitlements = useCallback(async () => {
    try {
      const keyword = entUser.trim()
      const params: {
        username?: string
        user_id?: number
        status?: string
      } = { status: 'active' }
      if (keyword) {
        if (/^\d+$/.test(keyword)) {
          params.user_id = Number(keyword)
        } else {
          params.username = keyword
        }
      }
      const list = await fetchByokEntitlements(params)
      setEntitlements(list)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to load entitlements')
      )
    }
  }, [entUser, t])

  useEffect(() => {
    if (!isRoot) return
    void refresh()
    void refreshEntitlements()
  }, [isRoot, refresh, refreshEntitlements])

  if (!isRoot) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('BYOK Providers')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <p className='text-muted-foreground text-sm'>
            {t('Only root can manage BYOK providers.')}
          </p>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const openCreate = () => {
    setCreating(true)
    setEditing(emptyForm())
  }

  const openEdit = (row: ByokProviderAdmin) => {
    setCreating(false)
    setEditing({
      id: row.id,
      name: row.name,
      channel_type: row.channel_type,
      models: row.models || '',
      key_hint: row.key_hint || '',
      need_base: row.need_base,
      base_hint: row.base_hint || '',
      other: row.other || '',
      key_url: row.key_url || '',
      logo_url: row.logo_url || '',
      enabled: row.enabled,
      sort: row.sort,
      is_openai_compatible: row.is_openai_compatible,
      pricing_tier: row.pricing_tier === 'paid' ? 'paid' : 'free',
    })
  }

  const paidProviders = items.filter((row) => row.pricing_tier === 'paid')

  const onGrant = async () => {
    const keyword = entUser.trim()
    const providerId = (entProvider || paidProviders[0]?.id || '').trim()
    if (!keyword || !providerId) {
      toast.error(t('User and paid provider are required'))
      return
    }
    try {
      setEntBusy(true)
      const payload: {
        username?: string
        user_id?: number
        provider_id: string
        note?: string
      } = {
        provider_id: providerId,
        note: entNote.trim() || undefined,
      }
      if (/^\d+$/.test(keyword)) {
        payload.user_id = Number(keyword)
      } else {
        payload.username = keyword
      }
      await grantByokEntitlement(payload)
      toast.success(t('Entitlement granted'))
      setEntNote('')
      await refreshEntitlements()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to grant entitlement')
      )
    } finally {
      setEntBusy(false)
    }
  }

  const onRevoke = async (row: ByokEntitlement) => {
    try {
      setEntBusy(true)
      await revokeByokEntitlement(row.id)
      toast.success(t('Entitlement revoked'))
      await refreshEntitlements()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to revoke entitlement')
      )
    } finally {
      setEntBusy(false)
    }
  }

  const onSave = async () => {
    if (!editing) return
    try {
      setSaving(true)
      if (creating) {
        await createByokProvider(editing)
        toast.success(t('Provider created'))
      } else {
        await updateByokProvider(editing.id, editing)
        toast.success(t('Provider saved'))
      }
      setEditing(null)
      await refresh()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to save provider')
      )
    } finally {
      setSaving(false)
    }
  }

  const onToggle = async (row: ByokProviderAdmin, enabled: boolean) => {
    try {
      await setByokProviderEnabled(row.id, enabled)
      setItems((prev) =>
        prev.map((item) => (item.id === row.id ? { ...item, enabled } : item))
      )
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to update provider')
      )
    }
  }

  const move = async (index: number, delta: number) => {
    const next = index + delta
    if (next < 0 || next >= items.length) return
    const ordered = [...items]
    const tmp = ordered[index]
    ordered[index] = ordered[next]
    ordered[next] = tmp
    setItems(ordered)
    try {
      const list = await reorderByokProviders(ordered.map((item) => item.id))
      setItems(list)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to reorder providers')
      )
      void refresh()
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('BYOK Providers')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <p className='text-muted-foreground mb-4 text-sm'>
          {t(
            'Configure the vendor catalog for user bring-your-own keys. This is not the shared channel pool.'
          )}
        </p>
        <div className='mb-4 flex flex-wrap items-center gap-3'>
          <Button size='sm' onClick={openCreate}>
            <Plus className='mr-1 h-4 w-4' />
            {t('Add provider')}
          </Button>
          <Button variant='outline' size='sm' onClick={() => void refresh()}>
            {t('Refresh')}
          </Button>
        </div>

        {loading && items.length === 0 ? (
          <p className='text-muted-foreground text-sm'>{t('Loading…')}</p>
        ) : (
          <div className='overflow-x-auto rounded-lg border border-border'>
            <table className='w-full min-w-[720px] text-left text-sm'>
              <thead className='bg-muted/40 text-muted-foreground'>
                <tr>
                  <th className='px-3 py-2 font-medium'>{t('Sort')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Vendor')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Type')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Pricing')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Enabled')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {items.map((row, index) => (
                  <tr key={row.id} className='border-t border-border'>
                    <td className='px-3 py-2'>
                      <div className='flex items-center gap-1'>
                        <Button
                          variant='ghost'
                          size='icon'
                          className='h-7 w-7'
                          disabled={index === 0}
                          onClick={() => void move(index, -1)}
                        >
                          <ArrowUp className='h-3.5 w-3.5' />
                        </Button>
                        <Button
                          variant='ghost'
                          size='icon'
                          className='h-7 w-7'
                          disabled={index === items.length - 1}
                          onClick={() => void move(index, 1)}
                        >
                          <ArrowDown className='h-3.5 w-3.5' />
                        </Button>
                      </div>
                    </td>
                    <td className='px-3 py-2'>
                      <div className='flex items-center gap-2'>
                        <img
                          src={providerLogoSrc(row)}
                          alt=''
                          className='h-5 w-5 object-contain'
                        />
                        <div>
                          <div className='font-medium'>{row.name}</div>
                          <div className='text-muted-foreground text-xs'>
                            {row.id}
                            {row.is_openai_compatible ? (
                              <Badge variant='outline' className='ml-2'>
                                OpenAI-compatible
                              </Badge>
                            ) : null}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className='px-3 py-2'>{row.channel_type}</td>
                    <td className='px-3 py-2'>
                      <Badge
                        variant={
                          row.pricing_tier === 'paid' ? 'default' : 'outline'
                        }
                      >
                        {row.pricing_tier === 'paid' ? t('Paid') : t('Free')}
                      </Badge>
                    </td>
                    <td className='px-3 py-2'>
                      <Switch
                        checked={row.enabled}
                        onCheckedChange={(checked) =>
                          void onToggle(row, checked)
                        }
                      />
                    </td>
                    <td className='px-3 py-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openEdit(row)}
                      >
                        {t('Edit')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <Dialog
          open={editing !== null}
          onOpenChange={(open) => {
            if (!open) setEditing(null)
          }}
        >
          <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-lg'>
            {editing ? (
              <>
                <DialogHeader>
                  <DialogTitle>
                    {creating ? t('Add provider') : t('Edit provider')}
                  </DialogTitle>
                  <DialogDescription>
                    {t(
                      'Configure the vendor catalog for user bring-your-own keys. This is not the shared channel pool.'
                    )}
                  </DialogDescription>
                </DialogHeader>
                <div className='grid gap-3'>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('ID (slug)')}
                    </label>
                    <Input
                      value={editing.id}
                      disabled={!creating}
                      onChange={(e) =>
                        setEditing({
                          ...editing,
                          id: e.target.value.toLowerCase(),
                        })
                      }
                      placeholder='openai'
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Name')}
                    </label>
                    <Input
                      value={editing.name}
                      onChange={(e) =>
                        setEditing({ ...editing, name: e.target.value })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Channel type')}
                    </label>
                    <Input
                      type='number'
                      value={editing.channel_type}
                      onChange={(e) =>
                        setEditing({
                          ...editing,
                          channel_type: Number(e.target.value) || 0,
                        })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Default models')}
                    </label>
                    <Input
                      value={editing.models || ''}
                      onChange={(e) =>
                        setEditing({ ...editing, models: e.target.value })
                      }
                      placeholder='model-a,model-b'
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Key hint')}
                    </label>
                    <Input
                      value={editing.key_hint || ''}
                      onChange={(e) =>
                        setEditing({ ...editing, key_hint: e.target.value })
                      }
                    />
                  </div>
                  <div className='flex items-center justify-between gap-3'>
                    <span className='text-sm'>{t('Need Base URL')}</span>
                    <Switch
                      checked={Boolean(editing.need_base)}
                      onCheckedChange={(checked) =>
                        setEditing({ ...editing, need_base: checked })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Base URL hint')}
                    </label>
                    <Input
                      value={editing.base_hint || ''}
                      onChange={(e) =>
                        setEditing({ ...editing, base_hint: e.target.value })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Other (e.g. Azure API version)')}
                    </label>
                    <Input
                      value={editing.other || ''}
                      onChange={(e) =>
                        setEditing({ ...editing, other: e.target.value })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Key obtain URL')}
                    </label>
                    <Input
                      value={editing.key_url || ''}
                      onChange={(e) =>
                        setEditing({ ...editing, key_url: e.target.value })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Logo URL (optional)')}
                    </label>
                    <Input
                      value={editing.logo_url || ''}
                      onChange={(e) =>
                        setEditing({ ...editing, logo_url: e.target.value })
                      }
                      placeholder={`/logos/${editing.id || 'id'}.svg`}
                    />
                  </div>
                  <div className='flex items-center justify-between gap-3'>
                    <span className='text-sm'>{t('OpenAI compatible')}</span>
                    <Switch
                      checked={Boolean(editing.is_openai_compatible)}
                      onCheckedChange={(checked) =>
                        setEditing({
                          ...editing,
                          is_openai_compatible: checked,
                        })
                      }
                    />
                  </div>
                  <div>
                    <label className='mb-1 block text-sm font-medium'>
                      {t('Pricing tier')}
                    </label>
                    <select
                      className='border-input bg-background h-9 w-full rounded-md border px-3 text-sm'
                      value={editing.pricing_tier === 'paid' ? 'paid' : 'free'}
                      onChange={(e) =>
                        setEditing({
                          ...editing,
                          pricing_tier:
                            e.target.value === 'paid' ? 'paid' : 'free',
                        })
                      }
                    >
                      <option value='free'>{t('Free — users may connect immediately')}</option>
                      <option value='paid'>
                        {t('Paid — requires entitlement, then user own key')}
                      </option>
                    </select>
                    <p className='text-muted-foreground mt-1 text-xs'>
                      {t(
                        'Paid only locks whether users can connect. After activation they still use their own official API key — this site does not resell shared keys.'
                      )}
                    </p>
                  </div>
                  <div className='flex items-center justify-between gap-3'>
                    <span className='text-sm'>{t('Enabled')}</span>
                    <Switch
                      checked={editing.enabled !== false}
                      onCheckedChange={(checked) =>
                        setEditing({ ...editing, enabled: checked })
                      }
                    />
                  </div>
                  <div className='flex justify-end gap-2 pt-2'>
                    <Button
                      variant='outline'
                      onClick={() => setEditing(null)}
                      disabled={saving}
                    >
                      {t('Cancel')}
                    </Button>
                    <Button onClick={() => void onSave()} disabled={saving}>
                      {saving ? t('Saving...') : t('Save')}
                    </Button>
                  </div>
                </div>
              </>
            ) : null}
          </DialogContent>
        </Dialog>

        <div className='mt-10 border-t border-border pt-6'>
          <h2 className='mb-2 text-base font-semibold'>
            {t('Paid provider entitlements')}
          </h2>
          <p className='text-muted-foreground mb-4 text-sm'>
            {t(
              'Manually grant access to paid vendors. After activation the user still brings their own official API key; this site does not hold or resell shared keys.'
            )}
          </p>
          <div className='mb-4 flex flex-wrap items-end gap-3'>
            <div>
              <label className='mb-1 block text-xs font-medium'>
                {t('Username or user id')}
              </label>
              <Input
                value={entUser}
                onChange={(e) => setEntUser(e.target.value)}
                placeholder='alice / 12'
                className='w-44'
              />
            </div>
            <div>
              <label className='mb-1 block text-xs font-medium'>
                {t('Paid provider')}
              </label>
              <select
                className='border-input bg-background h-9 min-w-[10rem] rounded-md border px-3 text-sm'
                value={entProvider || paidProviders[0]?.id || ''}
                onChange={(e) => setEntProvider(e.target.value)}
                disabled={paidProviders.length === 0}
              >
                {paidProviders.length === 0 ? (
                  <option value=''>{t('No paid providers yet')}</option>
                ) : (
                  paidProviders.map((row) => (
                    <option key={row.id} value={row.id}>
                      {row.name} ({row.id})
                    </option>
                  ))
                )}
              </select>
            </div>
            <div>
              <label className='mb-1 block text-xs font-medium'>
                {t('Note')}
              </label>
              <Input
                value={entNote}
                onChange={(e) => setEntNote(e.target.value)}
                placeholder={t('manual grant')}
                className='w-48'
              />
            </div>
            <Button
              size='sm'
              disabled={entBusy || paidProviders.length === 0}
              onClick={() => void onGrant()}
            >
              {t('Grant')}
            </Button>
            <Button
              variant='outline'
              size='sm'
              disabled={entBusy}
              onClick={() => void refreshEntitlements()}
            >
              {t('Refresh')}
            </Button>
          </div>
          <div className='overflow-x-auto rounded-lg border border-border'>
            <table className='w-full min-w-[640px] text-left text-sm'>
              <thead className='bg-muted/40 text-muted-foreground'>
                <tr>
                  <th className='px-3 py-2 font-medium'>ID</th>
                  <th className='px-3 py-2 font-medium'>{t('User')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Provider')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Status')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Note')}</th>
                  <th className='px-3 py-2 font-medium'>{t('Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {entitlements.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className='text-muted-foreground px-3 py-4 text-sm'
                    >
                      {t('No active entitlements')}
                    </td>
                  </tr>
                ) : (
                  entitlements.map((row) => (
                    <tr key={row.id} className='border-t border-border'>
                      <td className='px-3 py-2'>{row.id}</td>
                      <td className='px-3 py-2'>{row.user_id}</td>
                      <td className='px-3 py-2'>{row.provider_id}</td>
                      <td className='px-3 py-2'>{row.status}</td>
                      <td className='px-3 py-2'>{row.note || '—'}</td>
                      <td className='px-3 py-2'>
                        {row.status === 'active' ? (
                          <Button
                            variant='outline'
                            size='sm'
                            disabled={entBusy}
                            onClick={() => void onRevoke(row)}
                          >
                            {t('Revoke')}
                          </Button>
                        ) : null}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
