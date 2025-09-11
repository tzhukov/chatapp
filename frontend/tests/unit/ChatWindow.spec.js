import { mount } from '@vue/test-utils';
import ChatWindow from '@/components/ChatWindow.vue';

describe('ChatWindow.vue', () => {
  const messages = [
    { user_id: 'alice', content: 'Hello', timestamp: new Date().toISOString() },
    { user_id: 'bob', content: 'Hi', timestamp: new Date().toISOString() }
  ];

  it('renders empty state when no messages', () => {
    const wrapper = mount(ChatWindow, { props: { messages: [] } });
    expect(wrapper.text()).toContain('No messages yet');
  });

  it('renders messages', () => {
    const wrapper = mount(ChatWindow, { props: { messages } });
    expect(wrapper.text()).toContain('Hello');
    expect(wrapper.text()).toContain('Hi');
  });
});
