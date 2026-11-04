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

import type {
  CreateSiteTokenPayload,
  FetchMyKeyModelsPayload,
  MyKeysState,
  SaveMyKeyPayload,
  SiteTokenResult,
  TestMyKeyPayload,
  TestMyKeyResult,
} from './types'

type ApiEnvelope<T> = {
  success: boolean
  message?: string
  data: T
}

export async function getMyKeys(): Promise<MyKeysState> {
  const response = await api.get<ApiEnvelope<MyKeysState>>('/api/my-keys/', {
    skipBusinessError: true,
    skipErrorHandler: true,
    disableDuplicate: true,
  })
  return requireServerSuccess(response.data).data
}

export async function saveMyKey(payload: SaveMyKeyPayload): Promise<void> {
  const response = await api.post<ApiEnvelope<unknown>>(
    '/api/my-keys/',
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  requireServerSuccess(response.data)
}

export async function deleteMyKey(provider: string): Promise<void> {
  const response = await api.delete<ApiEnvelope<unknown>>(
    `/api/my-keys/${encodeURIComponent(provider)}`,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  requireServerSuccess(response.data)
}

export async function fetchMyKeyModels(
  payload: FetchMyKeyModelsPayload
): Promise<string[]> {
  const response = await api.post<ApiEnvelope<string[]>>(
    '/api/my-keys/fetch_models',
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data ?? []
}

export async function testMyKey(
  payload: TestMyKeyPayload
): Promise<TestMyKeyResult> {
  const response = await api.post<ApiEnvelope<TestMyKeyResult>>(
    '/api/my-keys/test',
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
      disableDuplicate: true,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function createSiteToken(
  payload: CreateSiteTokenPayload
): Promise<SiteTokenResult> {
  const response = await api.post<ApiEnvelope<SiteTokenResult>>(
    '/api/my-keys/site-token',
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}
