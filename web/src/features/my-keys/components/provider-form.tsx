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
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useDebounce } from '@/hooks/use-debounce'

import { deleteMyKey, fetchMyKeyModels, saveMyKey, testMyKey } from '../api'
import type { ByokProvider, MyKeyLink, TestMyKeyResult } from '../types'

type ProviderFormProps = {
  provider: ByokProvider
  link?: MyKeyLink
  onDone: () => void
  onCancel: () => void
  /** When true, skip outer card chrome / title (Dialog supplies them). */
  embedded?: boolean
}

type ProbeStatus = 'idle' | 'testing' | 'ok' | 'fail' | 'skipped'

const MIN_SECRET_LEN = 8
const PROBE_DEBOUNCE_MS = 700

function normalizeModelsValue(value: string): string {
  return value
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
    .join(',')
}

export function ProviderForm({
  provider,
  link,
  onDone,
  onCancel,
  embedded = false,
}: ProviderFormProps) {
  const { t } = useTranslation()
  const defaultModels = provider.models || ''
  const [secret, setSecret] = useState('')
  const [models, setModels] = useState(link?.models || defaultModels)
  const [baseUrl, setBaseUrl] = useState(link?.base_url || '')
  const [saving, setSaving] = useState(false)
  const [fetching, setFetching] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [probeStatus, setProbeStatus] = useState<ProbeStatus>('idle')
  const [probeMessage, setProbeMessage] = useState('')
  const [probeResult, setProbeResult] = useState<TestMyKeyResult | null>(null)
  const [fillHintShown, setFillHintShown] = useState(false)

  const debouncedSecret = useDebounce(secret, PROBE_DEBOUNCE_MS)
  const debouncedBaseUrl = useDebounce(baseUrl, PROBE_DEBOUNCE_MS)
  const probeSeq = useRef(0)
  const lastProbeKey = useRef('')
  const modelsRef = useRef(models)
  modelsRef.current = models

  const connected = Boolean(link)

  function canProbe(secretValue: string, baseValue: string): boolean {
    const trimmed = secretValue.trim()
    if (trimmed.length < MIN_SECRET_LEN) {
      return false
    }
    if (provider.need_base && !baseValue.trim()) {
      return false
    }
    return true
  }

  async function runProbe(opts?: {
    secretValue?: string
    baseValue?: string
    force?: boolean
  }): Promise<{ ok: boolean; message: string }> {
    const secretValue = (opts?.secretValue ?? secret).trim()
    const baseValue = (opts?.baseValue ?? baseUrl).trim()
    if (!canProbe(secretValue, baseValue)) {
      const skippedMsg =
        provider.need_base && !baseValue
          ? t('Enter Base URL before testing')
          : ''
      setProbeStatus(provider.need_base && !baseValue ? 'skipped' : 'idle')
      setProbeMessage(skippedMsg)
      setProbeResult(null)
      return { ok: false, message: skippedMsg }
    }

    const probeKey = `${provider.id}|${secretValue}|${baseValue}`
    if (!opts?.force && probeKey === lastProbeKey.current && probeStatus === 'ok') {
      return { ok: true, message: probeMessage }
    }

    const seq = ++probeSeq.current
    setProbeStatus('testing')
    setProbeMessage('')
    try {
      const result = await testMyKey({
        provider: provider.id,
        secret: secretValue,
        base_url: baseValue || undefined,
        models: modelsRef.current.trim() || undefined,
      })
      if (seq !== probeSeq.current) {
        return { ok: false, message: '' }
      }
      lastProbeKey.current = probeKey
      setProbeResult(result)
      setProbeStatus('ok')
      const count = result.models_count ?? result.models?.length ?? 0
      const okMsg = t('Key reachable · {{count}} models', { count })
      setProbeMessage(okMsg)

      const currentModels = normalizeModelsValue(modelsRef.current)
      const defaults = normalizeModelsValue(defaultModels)
      const modelList = result.models?.length
        ? result.models
        : result.sample || []
      if (
        !fillHintShown &&
        modelList.length > 0 &&
        currentModels === defaults
      ) {
        setFillHintShown(true)
        toast.message(t('Vendor returned models — fill in with one click?'), {
          action: {
            label: t('Fill models'),
            onClick: () => {
              setModels(modelList.join(','))
            },
          },
        })
      }
      return { ok: true, message: okMsg }
    } catch (error) {
      if (seq !== probeSeq.current) {
        return { ok: false, message: '' }
      }
      lastProbeKey.current = probeKey
      const failMsg =
        error instanceof Error ? error.message : t('Key probe failed')
      setProbeResult(null)
      setProbeStatus('fail')
      setProbeMessage(failMsg)
      return { ok: false, message: failMsg }
    }
  }

  useEffect(() => {
    const secretValue = debouncedSecret.trim()
    const baseValue = debouncedBaseUrl.trim()
    if (!secretValue) {
      setProbeStatus('idle')
      setProbeMessage('')
      setProbeResult(null)
      lastProbeKey.current = ''
      return
    }
    if (!canProbe(secretValue, baseValue)) {
      setProbeStatus(provider.need_base && !baseValue ? 'skipped' : 'idle')
      setProbeMessage(
        provider.need_base && !baseValue
          ? t('Enter Base URL before testing')
          : ''
      )
      return
    }
    const probeKey = `${provider.id}|${secretValue}|${baseValue}`
    if (probeKey === lastProbeKey.current) {
      return
    }
    void runProbe({ secretValue, baseValue })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- debounce-driven probe
  }, [debouncedSecret, debouncedBaseUrl, provider.id, provider.need_base])

  async function handleSave() {
    if (!connected && !secret.trim()) {
      toast.error(t('Please enter the vendor API key'))
      return
    }
    if (!models.trim()) {
      toast.error(t('Please enter at least one model name'))
      return
    }
    if (provider.need_base && provider.id === 'azure' && !baseUrl.trim() && !connected) {
      toast.error(t('Please enter the Azure resource URL'))
      return
    }

    if (secret.trim() && canProbe(secret, baseUrl)) {
      if (probeStatus === 'testing') {
        toast.error(t('Still testing the key, please wait'))
        return
      }
      // idle / fail / skipped：保存前强制再验；已 ok 可复用最近结果
      if (probeStatus !== 'ok') {
        const probed = await runProbe({ force: true })
        if (!probed.ok) {
          toast.error(
            probed.message ||
              t('Key probe failed — fix the key before saving')
          )
          return
        }
      }
    }

    try {
      setSaving(true)
      await saveMyKey({
        provider: provider.id,
        secret: secret.trim() || undefined,
        models: models.trim(),
        base_url: baseUrl.trim() || undefined,
      })
      toast.success(t('Vendor key saved'))
      onDone()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to save vendor key')
      )
    } finally {
      setSaving(false)
    }
  }

  async function handleFetchModels() {
    if (!secret.trim()) {
      toast.error(t('Re-enter the plaintext key to fetch models'))
      return
    }
    try {
      setFetching(true)
      const list = await fetchMyKeyModels({
        provider: provider.id,
        secret: secret.trim(),
        base_url: baseUrl.trim() || undefined,
      })
      if (!list.length) {
        toast.error(t('No models returned by the vendor'))
        return
      }
      setModels(list.join(','))
      toast.success(t('Fetched {{count}} models', { count: list.length }))
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to fetch models')
      )
    } finally {
      setFetching(false)
    }
  }

  async function handleDelete() {
    try {
      setDeleting(true)
      await deleteMyKey(provider.id)
      toast.success(t('Vendor key removed'))
      onDone()
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to delete vendor key')
      )
    } finally {
      setDeleting(false)
    }
  }

  function handleSecretBlur() {
    const secretValue = secret.trim()
    const baseValue = baseUrl.trim()
    if (!canProbe(secretValue, baseValue)) {
      return
    }
    const probeKey = `${provider.id}|${secretValue}|${baseValue}`
    if (probeKey === lastProbeKey.current && probeStatus !== 'idle') {
      return
    }
    void runProbe({ secretValue, baseValue, force: true })
  }

  return (
    <div
      className={
        embedded
          ? 'space-y-4'
          : 'space-y-4 rounded-xl border border-border bg-card p-4'
      }
    >
      {embedded ? null : (
        <div>
          <h3 className='flex items-center gap-2 text-base font-semibold'>
            <img
              src={(provider.logo_url || '').trim() || `/logos/${provider.id}.svg`}
              alt=''
              className='h-6 w-6 shrink-0 object-contain'
            />
            <span>{provider.name}</span>
          </h3>
          <p className='flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground'>
            <span>
              {connected
                ? t('Connected · {{hint}}', { hint: link?.key_hint || '••••' })
                : t('Not connected yet')}
            </span>
            {provider.key_url ? (
              <a
                href={provider.key_url}
                target='_blank'
                rel='noopener noreferrer'
                className='font-medium text-primary underline-offset-4 hover:underline'
              >
                {t('Get API Key')}
              </a>
            ) : null}
          </p>
        </div>
      )}

      <div className='space-y-2'>
        <Label htmlFor={`secret-${provider.id}`}>
          {t('Official vendor API key')}
        </Label>
        <Input
          id={`secret-${provider.id}`}
          type='password'
          autoComplete='off'
          placeholder={
            connected
              ? t('Leave blank to keep the saved key ({{hint}})', {
                  hint: provider.key_hint,
                })
              : provider.key_hint
          }
          value={secret}
          onChange={(event) => {
            setSecret(event.target.value)
            setFillHintShown(false)
            if (probeStatus === 'ok' || probeStatus === 'fail') {
              setProbeStatus('idle')
              setProbeMessage('')
            }
          }}
          onBlur={handleSecretBlur}
        />
        {probeStatus !== 'idle' ? (
          <p
            className={
              probeStatus === 'ok'
                ? 'text-sm text-emerald-600 dark:text-emerald-400'
                : probeStatus === 'fail'
                  ? 'text-sm text-destructive'
                  : 'text-sm text-muted-foreground'
            }
            role='status'
          >
            {probeStatus === 'testing'
              ? t('Testing key…')
              : probeStatus === 'ok'
                ? probeMessage || t('Key reachable')
                : probeStatus === 'skipped'
                  ? probeMessage
                  : probeMessage || t('Key probe failed')}
            {probeStatus === 'ok' && probeResult?.sample?.length ? (
              <span className='ml-1 text-muted-foreground'>
                ({probeResult.sample.slice(0, 3).join(', ')}
                {(probeResult.models_count || 0) > 3 ? '…' : ''})
              </span>
            ) : null}
          </p>
        ) : null}
      </div>

      {provider.need_base ? (
        <div className='space-y-2'>
          <Label htmlFor={`base-${provider.id}`}>{t('Base URL')}</Label>
          <Input
            id={`base-${provider.id}`}
            placeholder={provider.base_hint || 'https://...'}
            value={baseUrl}
            onChange={(event) => {
              setBaseUrl(event.target.value)
              setFillHintShown(false)
              if (probeStatus === 'ok' || probeStatus === 'fail') {
                setProbeStatus('idle')
                setProbeMessage('')
              }
            }}
            onBlur={handleSecretBlur}
          />
        </div>
      ) : null}

      <div className='space-y-2'>
        <Label htmlFor={`models-${provider.id}`}>
          {t('Models (comma separated)')}
        </Label>
        <Textarea
          id={`models-${provider.id}`}
          rows={3}
          value={models}
          onChange={(event) => setModels(event.target.value)}
        />
      </div>

      <div className='flex flex-wrap items-center gap-2'>
        <Button onClick={handleSave} disabled={saving || probeStatus === 'testing'}>
          {saving ? t('Saving…') : t('Save')}
        </Button>
        <span
          className={
            probeStatus === 'ok'
              ? 'text-sm text-emerald-600 dark:text-emerald-400'
              : probeStatus === 'fail'
                ? 'text-sm text-destructive'
                : 'text-sm text-muted-foreground'
          }
          role='status'
        >
          {probeStatus === 'testing'
            ? t('Testing key…')
            : probeStatus === 'ok'
              ? probeMessage || t('Key reachable')
              : probeStatus === 'fail'
                ? probeMessage || t('Key probe failed')
                : probeStatus === 'skipped'
                  ? probeMessage || t('Enter Base URL before testing')
                  : secret.trim()
                    ? canProbe(secret, baseUrl)
                      ? t('Not verified yet — will check on save')
                      : t('Enter a valid key to verify')
                    : connected
                      ? t('Keeping saved key · leave blank to skip re-check')
                      : t('Enter key to verify before save')}
        </span>
        <Button
          variant='outline'
          onClick={handleFetchModels}
          disabled={fetching}
        >
          {fetching ? t('Fetching…') : t('Fetch models from vendor')}
        </Button>
        {connected ? (
          <Button
            variant='destructive'
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? t('Deleting…') : t('Remove')}
          </Button>
        ) : null}
        <Button variant='ghost' onClick={onCancel}>
          {t('Cancel')}
        </Button>
      </div>
    </div>
  )
}
