const {
  window: { localStorage },
} = global;

const local = {
  clear: (): void => {
    localStorage.clear();
  },
  getItem: (itemName: string): string | null => {
    return localStorage.getItem(`MDMLAB::${itemName}`);
  },
  setItem: (itemName: string, value: string): void => {
    return localStorage.setItem(`MDMLAB::${itemName}`, value);
  },
  removeItem: (itemName: string): void => {
    localStorage.removeItem(`MDMLAB::${itemName}`);
  },
};

export const authToken = (): string | null => {
  return local.getItem("auth_token");
};

export const clearToken = (): void => {
  return local.removeItem("auth_token");
};

export default local;
