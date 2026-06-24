import { defineStore } from 'pinia';

export const useAppStore = defineStore('app', {
  state: () => ({
    collapsed: false,
    account: localStorage.getItem('dw0rdwk_account') || '',
    role: localStorage.getItem('dw0rdwk_role') || '',
  }),
  actions: {
    toggleCollapsed() {
      this.collapsed = !this.collapsed;
    },
    setAccount(account: string) {
      this.account = account;
      if (account) {
        localStorage.setItem('dw0rdwk_account', account);
      } else {
        localStorage.removeItem('dw0rdwk_account');
      }
    },
    setRole(role: string) {
      this.role = role;
      if (role) {
        localStorage.setItem('dw0rdwk_role', role);
      } else {
        localStorage.removeItem('dw0rdwk_role');
      }
    },
  },
});
