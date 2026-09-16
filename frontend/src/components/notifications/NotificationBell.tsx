import React, { useState, useEffect, useRef } from 'react';
import { Bell, CheckCheck, AlertTriangle, Clock, FileText, Info } from 'lucide-react';
import { notificationsApi } from '../../api/notifications';
import { Notification } from '../../types';
import { formatDateTime } from '../../lib/utils';
import { useAuth } from '../../context/AuthContext';

export const NotificationBell: React.FC = () => {
  const { user } = useAuth();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const fetchNotifications = async () => {
    if (!user || user.role !== 'user') return;
    try {
      const data = await notificationsApi.getMyNotifications();
      setNotifications(data || []);
    } catch (e) {
      console.warn('Failed to fetch notifications:', e);
    }
  };

  useEffect(() => {
    if (user && user.role === 'user') {
      fetchNotifications();
      const interval = setInterval(fetchNotifications, 15000);
      return () => clearInterval(interval);
    }
  }, [user]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const unreadCount = notifications.filter((n) => !n.is_read).length;

  const handleMarkAsRead = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await notificationsApi.markAsRead(id);
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, is_read: true } : n))
      );
    } catch (err) {
      console.error(err);
    }
  };

  const handleMarkAllRead = async () => {
    setLoading(true);
    try {
      await notificationsApi.markAllAsRead();
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (!user || user.role !== 'user') {
    return null;
  }

  const getIcon = (type: Notification['type']) => {
    switch (type) {
      case 'construction_delay':
        return <Clock className="w-3.5 h-3.5 text-zinc-900" />;
      case 'construction_risk':
        return <AlertTriangle className="w-3.5 h-3.5 text-amber-600" />;
      case 'deal_update':
        return <FileText className="w-3.5 h-3.5 text-zinc-900" />;
      default:
        return <Info className="w-3.5 h-3.5 text-zinc-900" />;
    }
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="relative w-8 h-8 rounded-xl border border-zinc-300 hover:border-zinc-900 hover:bg-[#FAF8F2] flex items-center justify-center text-zinc-600 hover:text-zinc-900 transition-colors cursor-pointer"
        title="Уведомления"
      >
        <Bell className="w-3.5 h-3.5" />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 flex items-center justify-center min-w-[16px] h-[16px] px-1 text-[9px] font-extrabold text-white bg-zinc-900 rounded-full border border-white">
            {unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-80 sm:w-96 bg-white rounded-2xl border-2 border-zinc-900 p-4 shadow-xl z-50">
          <div className="flex items-center justify-between pb-3 border-b border-zinc-200">
            <div className="flex items-center gap-2">
              <h3 className="font-extrabold text-zinc-900 uppercase text-xs tracking-wider">Уведомления</h3>
              {unreadCount > 0 && (
                <span className="px-2 py-0.2 rounded-full text-[10px] font-extrabold bg-[#FEF7EE] text-amber-950 border border-zinc-900">
                  +{unreadCount}
                </span>
              )}
            </div>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                disabled={loading}
                className="text-[11px] text-zinc-700 hover:text-zinc-900 hover:underline font-bold flex items-center gap-1 cursor-pointer"
              >
                <CheckCheck className="w-3.5 h-3.5" />
                Все прочитаны
              </button>
            )}
          </div>

          <div className="max-h-[360px] overflow-y-auto space-y-2 mt-3 pr-1">
            {notifications.length === 0 ? (
              <div className="py-6 text-center text-zinc-500 font-medium text-xs">
                Нет новых уведомлений
              </div>
            ) : (
              notifications.map((item) => (
                <div
                  key={item.id}
                  onClick={(e) => !item.is_read && handleMarkAsRead(item.id, e)}
                  className={`p-3 rounded-xl border transition-all cursor-pointer flex gap-3 ${
                    !item.is_read ? 'bg-[#FAF8F2] border-zinc-900 shadow-xs' : 'bg-white border-zinc-200 hover:border-zinc-400'
                  }`}
                >
                  <div className="mt-0.5 p-1.5 bg-white border border-zinc-300 rounded-lg shrink-0 h-fit">
                    {getIcon(item.type)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-extrabold text-zinc-900 leading-tight">{item.title}</p>
                    <p className="text-[11px] text-zinc-600 mt-1 leading-snug">{item.message}</p>
                    <span className="text-[10px] text-zinc-400 font-semibold mt-1.5 block">
                      {formatDateTime(item.created_at)}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};
