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

export type ByokVendorPrice = {
  provider: string
  model: string
  input_cny: string
  output_cny: string
  unit: string
  currency: string
  source: string
  updated_at: number
}

type ApiEnvelope<T> = {
  success: boolean
  message?: string
  data: T
}

export async function fetchByokPrices(provider = '') {
  const response = await api.get<
    ApiEnvelope<{
      items: ByokVendorPrice[]
      vendors: Record<string, string>
      count: number
    }>
  >('/api/byok/prices', {
    params: provider ? { provider } : undefined,
    skipBusinessError: true,
    skipErrorHandler: true,
    disableDuplicate: true,
  })
  return requireServerSuccess(response.data).data
}

export async function syncByokPrices(provider = 'all') {
  const response = await api.post<ApiEnvelope<Record<string, unknown>>>(
    '/api/byok/prices/sync',
    { provider },
    {
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  return requireServerSuccess(response.data).data
}
