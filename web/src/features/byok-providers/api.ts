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
import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export type ByokPricingTier = 'free' | 'paid'

export type ByokProviderAdmin = {
  id: string
  name: string
  channel_type: number
  models: string
  key_hint: string
  need_base: boolean
  base_hint?: string
  other?: string
  key_url?: string
  logo_url?: string
  enabled: boolean
  sort: number
  is_openai_compatible: boolean
  pricing_tier: ByokPricingTier
  updated_at: number
}

export type ByokProviderInput = {
  id: string
  name: string
  channel_type: number
  models?: string
  key_hint?: string
  need_base?: boolean
  base_hint?: string
  other?: string
  key_url?: string
  logo_url?: string
  enabled?: boolean
  sort?: number
  is_openai_compatible?: boolean
  pricing_tier?: ByokPricingTier
}

export type ByokEntitlement = {
  id: number
  user_id: number
  provider_id: string
  status: string
  granted_by: number
  granted_at: number
  revoked_at: number
  note?: string
}

type ApiEnvelope<T> = {
  success: boolean
  message?: string
  data: T
}

export async function fetchByokProvidersAdmin() {
  const response = await api.get<
    ApiEnvelope<{ items: ByokProviderAdmin[]; note?: string }>
  >('/api/byok/providers', {
    skipBusinessError: true,
    skipErrorHandler: true,
    disableDuplicate: true,
  })
  return requireServerSuccess(response.data).data.items || []
}

export async function createByokProvider(payload: ByokProviderInput) {
  const response = await api.post<ApiEnvelope<ByokProviderAdmin>>(
    '/api/byok/providers',
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function updateByokProvider(
  id: string,
  payload: ByokProviderInput
) {
  const response = await api.put<ApiEnvelope<ByokProviderAdmin>>(
    `/api/byok/providers/${encodeURIComponent(id)}`,
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function setByokProviderEnabled(id: string, enabled: boolean) {
  const response = await api.put<ApiEnvelope<ByokProviderAdmin>>(
    `/api/byok/providers/${encodeURIComponent(id)}/enabled`,
    { enabled },
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function reorderByokProviders(ids: string[]) {
  const response = await api.put<
    ApiEnvelope<{ items: ByokProviderAdmin[] }>
  >(
    '/api/byok/providers/sort',
    { ids },
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data.items || []
}

export async function fetchByokEntitlements(params?: {
  user_id?: number
  username?: string
  provider_id?: string
  status?: string
}) {
  const response = await api.get<
    ApiEnvelope<{ items: ByokEntitlement[]; note?: string }>
  >('/api/byok/entitlements', {
    params,
    skipBusinessError: true,
    skipErrorHandler: true,
    disableDuplicate: true,
  })
  return requireServerSuccess(response.data).data.items || []
}

export async function grantByokEntitlement(payload: {
  user_id?: number
  username?: string
  provider_id: string
  note?: string
}) {
  const response = await api.post<ApiEnvelope<ByokEntitlement>>(
    '/api/byok/entitlements',
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function revokeByokEntitlement(id: number) {
  const response = await api.put<ApiEnvelope<ByokEntitlement>>(
    `/api/byok/entitlements/${id}/revoke`,
    {},
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}
