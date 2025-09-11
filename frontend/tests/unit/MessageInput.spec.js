import { mount } from '@vue/test-utils';
import MessageInput from '@/components/MessageInput.vue';

describe('MessageInput.vue', () => {
  it('emits send-message with trimmed value and clears input', async () => {
    const wrapper = mount(MessageInput);
    const input = wrapper.find('input');
    await input.setValue('  Hello ');
    await wrapper.find('button').trigger('click');
    const events = wrapper.emitted('send-message');
    expect(events).toBeTruthy();
    expect(events[0][0]).toBe('Hello');
    expect(input.element.value).toBe('');
  });

  it('does not emit when input is empty or whitespace', async () => {
    const wrapper = mount(MessageInput);
    await wrapper.find('button').trigger('click');
    expect(wrapper.emitted('send-message')).toBeUndefined();
    const input = wrapper.find('input');
    await input.setValue('   ');
    await wrapper.find('button').trigger('click');
    expect(wrapper.emitted('send-message')).toBeUndefined();
  });
});
