import React, { useState, useEffect } from 'react';
import { RefreshCw, Search, Filter } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { chatsApi } from '../api/chats';
import { apartmentsApi } from '../api/apartments';
import { ChatSession, ChatSessionStatus, Apartment } from '../types';
import { ChatRoom } from '../components/chat/ChatRoom';
import { formatPrice, formatDate, getChatSessionStatusBadge } from '../lib/utils';

export const ManagerDashboardPage: React.FC = () => {
  const { user } = useAuth();

  const [allSessions, setAllSessions] = useState<ChatSession[]>([]);
  const [apartmentsMap, setApartmentsMap] = useState<Record<number, Apartment>>({});

  const [activeTab, setActiveTab] = useState<'my' | 'queue'>('my');
  const [statusFilter, setStatusFilter] = useState<'all' | ChatSessionStatus>('all');
  const [activeSessionId, setActiveSessionId] = useState<number | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);

  const loadManagerData = async () => {
    try {
      const sessionsData = await chatsApi.getAllSessions().catch(() => []);
      setAllSessions(sessionsData);

      const aptIds = Array.from(new Set(sessionsData.map((s) => s.id_apartment).filter(Boolean))) as number[];
      const map: Record<number, Apartment> = {};
      for (const id of aptIds) {
        try {
          map[id] = await apartmentsApi.getApartment(id);
        } catch {}
      }
      setApartmentsMap(map);

      setActiveSessionId((prev) => {
        if (prev !== null) return prev;
        if (sessionsData.length > 0) {
          const myFirst = sessionsData.find((s) => s.id_employee === user?.id);
          return myFirst ? myFirst.id : sessionsData[0].id;
        }
        return null;
      });
    } catch (err) {
      console.error('Failed to load manager workspace data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadManagerData();
    const interval = setInterval(loadManagerData, 8000);
    return () => clearInterval(interval);
  }, [user]);

  const mySessions = allSessions.filter((s) => s.id_employee === user?.id);
  const queueSessions = allSessions.filter((s) => !s.id_employee || s.status === 'open');

  const currentTabSessions = activeTab === 'my' ? mySessions : queueSessions;

  const STATUS_OPTIONS: { id: 'all' | ChatSessionStatus; label: string }[] =
    activeTab === 'my'
      ? [
          { id: 'all', label: 'Все' },
          { id: 'in_progress', label: 'В работе' },
          { id: 'pending_approval', label: 'На согласовании' },
          { id: 'contract', label: 'Согласовано' },
          { id: 'close', label: 'Завершено' },
        ]
      : [
          { id: 'all', label: 'Все' },
          { id: 'open', label: 'Ожидают' },
        ];

  const visibleSessions = currentTabSessions.filter((s) => {

    if (statusFilter !== 'all' && s.status !== statusFilter) {
      return false;
    }

    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    const apt = s.id_apartment ? apartmentsMap[s.id_apartment] : null;
    const clientName = (s.user_name || s.guest_name || '').toLowerCase();
    return (
      clientName.includes(q) ||
      (apt && apt.number.includes(q)) ||
      s.id.toString().includes(q)
    );
  });

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 space-y-4 h-[calc(100vh-5rem)] flex flex-col overflow-hidden">
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 flex-1 min-h-0">

        <div className="lg:col-span-4 flex flex-col min-h-0 space-y-3">

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-4 flex items-center justify-between gap-3 shadow-xs shrink-0">
            <div className="flex items-center gap-3 min-w-0">
              <div className="w-11 h-11 rounded-2xl bg-[#FAF8F2] border-2 border-zinc-900 flex items-center justify-center font-extrabold text-sm text-zinc-900 shadow-xs shrink-0">
                {user?.name ? user.name.slice(0, 2).toUpperCase() : 'МГ'}
              </div>
              <div className="min-w-0">
                <h1 className="text-sm font-extrabold text-zinc-900 truncate">{user?.name || 'Менеджер'}</h1>
                <p className="text-[11px] text-zinc-500 font-medium truncate">{user?.email}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-4 flex flex-col flex-1 min-h-0 shadow-xs space-y-3">

            <div className="flex bg-[#FAF8F2] p-1 rounded-2xl border border-zinc-300 text-xs font-bold shrink-0">
              <button
                onClick={() => {
                  setActiveTab('my');
                  setStatusFilter('all');
                }}
                className={`flex-1 py-2 rounded-xl transition-all flex items-center justify-center gap-1.5 cursor-pointer ${
                  activeTab === 'my' ? 'bg-zinc-900 text-white shadow-xs' : 'text-zinc-600 hover:text-zinc-900'
                }`}
              >
                Мои заявки ({mySessions.length})
              </button>
              <button
                onClick={() => {
                  setActiveTab('queue');
                  setStatusFilter('all');
                }}
                className={`flex-1 py-2 rounded-xl transition-all flex items-center justify-center gap-1.5 cursor-pointer ${
                  activeTab === 'queue' ? 'bg-zinc-900 text-white shadow-xs' : 'text-zinc-600 hover:text-zinc-900'
                }`}
              >
                Ожидают ({queueSessions.length})
              </button>
            </div>

            <div className="flex items-center gap-1.5 shrink-0 overflow-x-auto pb-1 text-[11px] font-bold">
              {STATUS_OPTIONS.map((opt) => {
                const count =
                  opt.id === 'all'
                    ? currentTabSessions.length
                    : currentTabSessions.filter((s) => s.status === opt.id).length;

                return (
                  <button
                    key={opt.id}
                    onClick={() => setStatusFilter(opt.id)}
                    className={`px-2.5 py-1 rounded-lg border transition-colors whitespace-nowrap cursor-pointer ${
                      statusFilter === opt.id
                        ? 'bg-zinc-900 text-white border-zinc-900 shadow-xs'
                        : 'bg-[#FAF8F2] text-zinc-700 border-zinc-300 hover:border-zinc-900'
                    }`}
                  >
                    {opt.label} {count > 0 ? `(${count})` : ''}
                  </button>
                );
              })}
            </div>

            <div className="relative shrink-0">
              <Search className="w-4 h-4 text-zinc-400 absolute left-3.5 top-3" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Поиск по клиенту или квартире..."
                className="w-full pl-10 pr-3.5 py-2 bg-[#FAF8F2] border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-white"
              />
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto space-y-2.5 pr-1">
              {loading ? (
                <div className="py-8 text-center text-zinc-400">
                  <RefreshCw className="w-5 h-5 animate-spin mx-auto text-zinc-900" />
                </div>
              ) : visibleSessions.length === 0 ? (
                <div className="py-8 text-center text-zinc-500 text-xs font-semibold">
                  {activeTab === 'my' ? 'Нет обращений по выбранному фильтру.' : 'Очередь новых обращений пуста.'}
                </div>
              ) : (
                visibleSessions.map((sess) => {
                  const apt = sess.id_apartment ? apartmentsMap[sess.id_apartment] : null;
                  const isActive = activeSessionId === sess.id;
                  const isUnanswered = sess.status === 'open' || !sess.id_employee;
                  const clientFullName = sess.user_name || sess.guest_name || 'Клиент';
                  const badge = getChatSessionStatusBadge(sess.status);

                  return (
                    <div
                      key={sess.id}
                      onClick={() => setActiveSessionId(sess.id)}
                      className={`p-3 rounded-2xl border-2 transition-all cursor-pointer relative ${
                        isActive ? 'bg-[#FAF8F2] border-zinc-900 shadow-xs' : 'bg-white border-zinc-300 hover:border-zinc-900'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-1 mb-1">
                        <div className="flex items-center gap-1.5 truncate">
                          {isUnanswered && (
                            <span className="w-2 h-2 rounded-full bg-amber-500 shrink-0" title="Требует ответа" />
                          )}
                          <span className="text-xs font-extrabold text-zinc-900 truncate">{clientFullName}</span>
                        </div>
                        <span className="text-[10px] text-zinc-400 font-semibold shrink-0">{formatDate(sess.updated_at)}</span>
                      </div>

                      {apt ? (
                        <p className="text-[11px] text-zinc-600 font-medium truncate">
                          Кв. №{apt.number} • {apt.rooms}к, {apt.area} м² ({formatPrice(apt.price)})
                        </p>
                      ) : (
                        <p className="text-[11px] text-zinc-400">Общее обращение</p>
                      )}

                      <div className="mt-2 flex items-center justify-between pt-1.5 border-t border-zinc-200">
                        {apt ? (
                          <span
                            className={`text-[10px] font-extrabold px-2 py-0.5 rounded-full border ${badge.color}`}
                          >
                            {badge.label}
                          </span>
                        ) : (
                          <span className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider">
                            Общий вопрос
                          </span>
                        )}
                        <span className="text-[11px] text-zinc-900 font-bold">Открыть →</span>
                      </div>
                    </div>
                  );
                })
              )}
            </div>

          </div>

        </div>

        <div className="lg:col-span-8 flex flex-col min-h-0 h-full">
          {activeSessionId ? (
            <ChatRoom
              sessionId={activeSessionId}
              onSessionUpdated={loadManagerData}
              onSessionTaken={(takenSession) => {
                setActiveTab('my');
                setActiveSessionId(takenSession.id);
                loadManagerData();
              }}
            />
          ) : (
            <div className="h-full flex flex-col items-center justify-center p-8 bg-white rounded-3xl border-2 border-zinc-900 text-center text-zinc-400 space-y-1 shadow-xs">
              <h3 className="text-sm font-extrabold text-zinc-900">Выберите диалог из списка</h3>
              <p className="text-xs text-zinc-500 max-w-sm">
                Выберите обращение для ответа клиенту, использования ассистента или оформления сделки.
              </p>
            </div>
          )}
        </div>

      </div>
    </div>
  );
};
