import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ShortcutModal from '../ui/ShortcutModal.vue'

describe('ShortcutModal', () => {
  it('does not render when open is false', () => {
    const wrapper = mount(ShortcutModal, { props: { open: false } })
    expect(wrapper.find('.shortcut-overlay').exists()).toBe(false)
  })

  it('renders when open is true', () => {
    const wrapper = mount(ShortcutModal, {
      props: { open: true },
      global: { stubs: { teleport: true } },
    })
    expect(wrapper.find('.shortcut-modal').exists()).toBe(true)
    expect(wrapper.text()).toContain('Keyboard Shortcuts')
  })

  it('displays all shortcut groups', () => {
    const wrapper = mount(ShortcutModal, {
      props: { open: true },
      global: { stubs: { teleport: true } },
    })
    expect(wrapper.text()).toContain('Navigation')
    expect(wrapper.text()).toContain('File Actions')
    expect(wrapper.text()).toContain('Global')
  })

  it('emits close when close button clicked', async () => {
    const wrapper = mount(ShortcutModal, {
      props: { open: true },
      global: { stubs: { teleport: true } },
    })
    await wrapper.find('.shortcut-modal__close').trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
