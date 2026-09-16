import { api } from './client';
import { User, Role } from '../types';

export interface LoginResponse {
  token: string;
  user: User;
}

export const authApi = {

  loginUser: async (email: string, password: string): Promise<LoginResponse> => {
    return api.post<LoginResponse>('/auth/login', { email, password }, false);
  },

  loginStaff: async (email: string, password: string): Promise<LoginResponse> => {
    return api.post<LoginResponse>('/auth/login', { email, password }, true);
  },

  registerUser: async (name: string, email: string, password: string): Promise<User> => {
    return api.post<User>('/auth/register', { name, email, password }, false);
  },

  registerStaff: async (name: string, email: string, password: string, role: Role): Promise<User> => {
    return api.post<User>('/auth/register', { name, email, password, role }, true);
  },

  getUserProfile: async (): Promise<User> => {
    return api.get<User>('/users/me', false);
  },

  updateUserProfile: async (data: { name?: string; email?: string; password?: string }): Promise<User> => {
    return api.put<User>('/users/me', data, false);
  },

  getStaffProfile: async (): Promise<User> => {
    return api.get<User>('/staff/profile', true);
  },

  logoutUser: async (): Promise<void> => {
    return api.post('/auth/logout', {}, false);
  },

  logoutStaff: async (): Promise<void> => {
    return api.post('/auth/logout', {}, true);
  },
};
