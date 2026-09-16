import { api } from './client';
import { Notification } from '../types';

export const notificationsApi = {

  getMyNotifications: async (unreadOnly = false): Promise<Notification[]> => {
    const query = unreadOnly ? '?unread=true' : '';
    return api.get<Notification[]>(`/notifications${query}`, false);
  },

  markAsRead: async (id: number): Promise<void> => {
    return api.put(`/notifications/${id}/read`, {}, false);
  },

  markAllAsRead: async (): Promise<void> => {
    return api.put('/notifications/read-all', {}, false);
  },
};
