import { ref, computed } from 'vue'

const currentLocale = ref(localStorage.getItem('anveesa-locale') || 'en')

const messages = {
  en: {
    'nav.dashboard': 'Dashboard',
    'nav.search': 'Search',
    'nav.shared': 'Shared Links',
    'nav.audit': 'Audit Log',
    'nav.jobs': 'Jobs',
    'nav.docs': 'Docs',
    'nav.activity': 'Activity',
    'nav.split': 'Split view',
    'nav.signout': 'Sign out',
    'nav.newConnection': 'New Connection',
    'nav.filterConnections': 'Filter connections…',
    'nav.management': 'Management',
    'common.loading': 'Loading...',
    'common.save': 'Save',
    'common.cancel': 'Cancel',
    'common.delete': 'Delete',
    'common.confirm': 'Confirm',
    'common.close': 'Close',
    'common.refresh': 'Refresh',
    'common.noData': 'No data available',
    'auth.login': 'Sign In',
    'auth.register': 'Create Account',
    'auth.username': 'Username',
    'auth.password': 'Password',
    'shortcuts.title': 'Keyboard Shortcuts',
    'shortcuts.navigation': 'Navigation',
    'shortcuts.fileActions': 'File Actions',
    'shortcuts.global': 'Global',
  },
  id: {
    'nav.dashboard': 'Dasbor',
    'nav.search': 'Pencarian',
    'nav.shared': 'Tautan Bersama',
    'nav.audit': 'Log Audit',
    'nav.jobs': 'Pekerjaan',
    'nav.docs': 'Dokumentasi',
    'nav.activity': 'Aktivitas',
    'nav.split': 'Tampilan terpisah',
    'nav.signout': 'Keluar',
    'nav.newConnection': 'Koneksi Baru',
    'nav.filterConnections': 'Cari koneksi…',
    'nav.management': 'Manajemen',
    'common.loading': 'Memuat...',
    'common.save': 'Simpan',
    'common.cancel': 'Batal',
    'common.delete': 'Hapus',
    'common.confirm': 'Konfirmasi',
    'common.close': 'Tutup',
    'common.refresh': 'Segarkan',
    'common.noData': 'Tidak ada data',
    'auth.login': 'Masuk',
    'auth.register': 'Buat Akun',
    'auth.username': 'Nama Pengguna',
    'auth.password': 'Kata Sandi',
    'shortcuts.title': 'Pintasan Keyboard',
    'shortcuts.navigation': 'Navigasi',
    'shortcuts.fileActions': 'Aksi File',
    'shortcuts.global': 'Global',
  },
}

export function useI18n() {
  const locale = computed({
    get: () => currentLocale.value,
    set: (v) => {
      currentLocale.value = v
      localStorage.setItem('anveesa-locale', v)
    },
  })

  const availableLocales = Object.keys(messages)

  function t(key) {
    return messages[currentLocale.value]?.[key] || messages.en[key] || key
  }

  return { locale, availableLocales, t }
}
