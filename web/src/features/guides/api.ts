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

export type GuideCard = {
  slug: string
  title: string
  description: string
  sort_order: number
  updated_at: number
}

export type GuideDetail = GuideCard & {
  body: string
}

type ApiEnvelope<T> = {
  success: boolean
  message?: string
  data: T
}

export async function fetchGuides() {
  const response = await api.get<ApiEnvelope<{ items: GuideCard[] }>>(
    '/api/byok/guides',
    {
      skipBusinessError: true,
      skipErrorHandler: true,
      disableDuplicate: true,
    }
  )
  return requireServerSuccess(response.data).data.items || []
}

export async function fetchGuide(slug: string) {
  const response = await api.get<ApiEnvelope<GuideDetail>>(
    `/api/byok/guides/${encodeURIComponent(slug)}`,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
      disableDuplicate: true,
    }
  )
  return requireServerSuccess(response.data).data
}

export async function saveGuide(
  slug: string,
  payload: { title: string; description: string; body: string }
) {
  const response = await api.put<ApiEnvelope<GuideDetail>>(
    `/api/byok/guides/${encodeURIComponent(slug)}`,
    payload,
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}
