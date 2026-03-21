import { describe, it, expect, beforeEach } from 'vitest'
import { useI18n } from '../useI18n.js'

describe('useI18n', () => {
  beforeEach(() => {
    localStorage.clear()
    const { locale } = useI18n()
    locale.value = 'en'
  })

  it('defaults to English locale', () => {
    const { locale } = useI18n()
    expect(locale.value).toBe('en')
  })

  it('translates known keys', () => {
    const { t } = useI18n()
    expect(t('nav.dashboard')).toBe('Dashboard')
    expect(t('common.save')).toBe('Save')
  })

  it('returns key if translation missing', () => {
    const { t } = useI18n()
    expect(t('some.missing.key')).toBe('some.missing.key')
  })

  it('switches locale', () => {
    const { locale, t } = useI18n()
    locale.value = 'id'
    expect(t('nav.dashboard')).toBe('Dasbor')
    expect(t('common.save')).toBe('Simpan')
  })

  it('persists locale to localStorage', () => {
    const { locale } = useI18n()
    locale.value = 'id'
    expect(localStorage.getItem('anveesa-locale')).toBe('id')
  })

  it('lists available locales', () => {
    const { availableLocales } = useI18n()
    expect(availableLocales).toContain('en')
    expect(availableLocales).toContain('id')
  })
})
