import { createContext, useContext, useState, type ReactNode } from 'react';
import { AdminClient } from '../api/client';

interface AuthContextValue {
  adminKey: string;
  client: AdminClient | null;
  login: (key: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue>({
  adminKey: '', client: null,
  login: () => {}, logout: () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [adminKey, setAdminKey] = useState(() => localStorage.getItem('adminKey') ?? '');

  const client = adminKey ? new AdminClient(adminKey) : null;

  function login(key: string) {
    localStorage.setItem('adminKey', key);
    setAdminKey(key);
  }

  function logout() {
    localStorage.removeItem('adminKey');
    setAdminKey('');
  }

  return (
    <AuthContext.Provider value={{ adminKey, client, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
