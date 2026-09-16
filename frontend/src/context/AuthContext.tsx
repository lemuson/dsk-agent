import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { User, Role } from '../types';
import { authApi } from '../api/auth';

interface AuthContextType {
  user: User | null;
  role: Role | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string, isStaff?: boolean) => Promise<User>;
  register: (name: string, email: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
  refreshProfile: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const LOCAL_STORAGE_USER_KEY = 'dsk_current_user';

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(() => {
    try {
      const saved = localStorage.getItem(LOCAL_STORAGE_USER_KEY);
      return saved ? JSON.parse(saved) : null;
    } catch {
      return null;
    }
  });
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const saveUser = (u: User | null) => {
    setUser(u);
    if (u) {
      localStorage.setItem(LOCAL_STORAGE_USER_KEY, JSON.stringify(u));
    } else {
      localStorage.removeItem(LOCAL_STORAGE_USER_KEY);
    }
  };

  const refreshProfile = async () => {
    if (!user) {
      setIsLoading(false);
      return;
    }
    try {
      if (user.role === 'user') {
        const u = await authApi.getUserProfile();
        saveUser(u);
      } else {
        const u = await authApi.getStaffProfile();
        saveUser(u);
      }
    } catch (e: any) {
      console.warn('Failed to verify user session:', e);

      if (e?.status === 404 || e?.status === 401 || e?.message?.includes('user not found') || e?.message?.includes('unauthorized')) {
        saveUser(null);
      }
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    refreshProfile();
  }, []);

  const login = async (email: string, password: string, isStaff = false): Promise<User> => {
    setIsLoading(true);
    try {
      const resp = isStaff
        ? await authApi.loginStaff(email, password)
        : await authApi.loginUser(email, password);

      saveUser(resp.user);
      return resp.user;
    } finally {
      setIsLoading(false);
    }
  };

  const register = async (name: string, email: string, password: string): Promise<User> => {
    setIsLoading(true);
    try {
      const newUser = await authApi.registerUser(name, email, password);

      const loginResp = await authApi.loginUser(email, password);
      saveUser(loginResp.user);
      return loginResp.user;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    setIsLoading(true);
    try {
      if (user?.role === 'user') {
        await authApi.logoutUser();
      } else if (user) {
        await authApi.logoutStaff();
      }
    } catch (e) {
      console.warn('Logout error:', e);
    } finally {
      saveUser(null);
      setIsLoading(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        role: user?.role || null,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
        refreshProfile,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
