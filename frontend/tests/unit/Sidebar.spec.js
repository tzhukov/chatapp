import { mount } from '@vue/test-utils';
import Sidebar from '@/components/Sidebar.vue';

describe('Sidebar.vue', () => {
  const baseChat = (id, name, minsAgo = 0, messages = []) => ({
    id,
    name,
    messages,
    unreadCount: 0,
    lastActivity: new Date(Date.now() - minsAgo * 60 * 1000)
  });

  it('sorts chats by lastActivity descending', () => {
    const chats = [
      baseChat(1, 'A', 10),
      baseChat(2, 'B', 1),
      baseChat(3, 'C', 30)
    ];
    const wrapper = mount(Sidebar, { props: { chats, activeChat: chats[0] } });
    const rendered = wrapper.vm.sortedChats.map(c => c.id);
    expect(rendered).toEqual([2,1,3]);
  });

  it('emits selectChat when a chat is clicked', async () => {
    const chats = [baseChat(1,'A',5)];
    const wrapper = mount(Sidebar, { props: { chats, activeChat: chats[0] } });
    await wrapper.find('[data-v-app]');
    await wrapper.findAll('[class~="cursor-pointer"]')[0].trigger('click');
    expect(wrapper.emitted('selectChat')).toBeTruthy();
  });
});
