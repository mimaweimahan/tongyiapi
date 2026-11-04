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

export type ByokProvider = {
  id: string
  name: string
  type: number
  models: string
  key_hint: string
  need_base: boolean
  base_hint?: string
  other?: string
  /** Official vendor console page to obtain an API key; omit when empty. */
  key_url?: string
  /** Optional remote logo; falls back to /logos/{id}.svg */
  logo_url?: string
  /** free | paid — paid requires entitlement before connecting. */
  pricing_tier?: 'free' | 'paid'
  /** false when paid and not entitled; UI must block key entry. */
  can_connect?: boolean
}

export type MyKeyLink = {
  provider: string
  channel_id: number
  name: string
  group: string
  models: string
  chat_models?: string[]
  base_url?: string
  key_hint: string
  status: number
}

export type MyKeysState = {
  providers: ByokProvider[]
  links: Record<string, MyKeyLink>
  group: string
}

export type SaveMyKeyPayload = {
  provider: string
  secret?: string
  models: string
  base_url?: string
}

export type FetchMyKeyModelsPayload = {
  provider: string
  secret: string
  base_url?: string
}

export type TestMyKeyPayload = {
  provider: string
  secret: string
  base_url?: string
  models?: string
}

export type TestMyKeyResult = {
  ok: boolean
  models_count?: number
  sample?: string[]
  models?: string[]
}

export type CreateSiteTokenPayload = {
  provider: string
  model: string
  name?: string
}

export type SiteTokenResult = {
  id: number
  name: string
  key: string
  model: string
  scope: string
  provider: string
  group: string
  base_url: string
  models?: string
}
