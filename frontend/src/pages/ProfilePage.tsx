import React, { useState, useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Download, RefreshCw, Trash2, FileText, CheckCircle2, Plus, MessageSquare } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { chatsApi } from '../api/chats';
import { dealsApi } from '../api/deals';
import { apartmentsApi } from '../api/apartments';
import { ChatSession, Deal, Offer, Apartment } from '../types';
import { ChatRoom } from '../components/chat/ChatRoom';
import { formatPrice, getDealStatusBadge, getChatSessionStatusBadge, formatDate } from '../lib/utils';

export const ProfilePage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [deals, setDeals] = useState<Deal[]>([]);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [apartmentsMap, setApartmentsMap] = useState<Record<number, Apartment>>({});

  const [activeSessionId, setActiveSessionId] = useState<number | null>(() => {
    const fromUrl = searchParams.get('session');
    return fromUrl ? Number(fromUrl) : null;
  });

  const [loading, setLoading] = useState(true);
  const [downloadingPdf, setDownloadingPdf] = useState<number | null>(null);
  const [creatingGeneralChat, setCreatingGeneralChat] = useState(false);

  const loadData = async () => {
    setLoading(true);
    try {
      const [userSessions, userDeals, userOffers] = await Promise.all([
        chatsApi.getMySessions().catch(() => []),
        dealsApi.getMyDeals().catch(() => []),
        dealsApi.getMyOffers().catch(() => []),
      ]);

      setSessions(userSessions);
      setDeals(userDeals);
      setOffers(userOffers);

      setActiveSessionId((prev) => {
        if (prev !== null) return prev;
        return userSessions.length > 0 ? userSessions[0].id : null;
      });

      const aptIds = Array.from(new Set(userSessions.map((s) => s.id_apartment).filter(Boolean))) as number[];
      const map: Record<number, Apartment> = {};
      for (const id of aptIds) {
        try {
          map[id] = await apartmentsApi.getApartment(id);
        } catch {

        }
      }
      setApartmentsMap(map);
    } catch (err) {
      console.error('Failed to load profile data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreateGeneralChat = async () => {

    const existing = sessions.find((s) => !s.id_apartment);
    if (existing) {
      setActiveSessionId(existing.id);
      return;
    }

    setCreatingGeneralChat(true);
    try {
      const newSession = await chatsApi.createSession({});
      setSessions((prev) => [newSession, ...prev]);
      setActiveSessionId(newSession.id);
    } catch (err: any) {
      console.error('Failed to create general chat:', err);
      alert(err.message || 'Ошибка создания диалога');
    } finally {
      setCreatingGeneralChat(false);
    }
  };

  const handleDeleteSession = async (e: React.MouseEvent, sessId: number) => {
    e.stopPropagation();
    if (!window.confirm('Вы уверены, что хотите удалить этот диалог из списка? Переписка будет скрыта из вашего кабинета.')) {
      return;
    }
    try {
      await chatsApi.deleteSession(sessId);
      setSessions((prev) => prev.filter((s) => s.id !== sessId));
      if (activeSessionId === sessId) {
        const remaining = sessions.filter((s) => s.id !== sessId);
        setActiveSessionId(remaining.length > 0 ? remaining[0].id : null);
      }
    } catch (err: any) {
      console.error('Failed to delete chat session:', err);
      alert(err.message || 'Ошибка удаления диалога');
    }
  };

  const handleDownloadPdf = async (offerId: number) => {
    setDownloadingPdf(offerId);
    try {
      const blob = await dealsApi.downloadOfferPdf(offerId, false);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `коммерческое_предложение_${offerId}.pdf`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to download PDF:', err);
      alert('PDF доступен только для сформированных предложений.');
    } finally {
      setDownloadingPdf(null);
    }
  };

  const filteredSessions = sessions;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 space-y-4 h-[calc(100vh-5rem)] flex flex-col overflow-hidden">

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 flex-1 min-h-0">

        <div className="lg:col-span-4 flex flex-col gap-4 min-h-0">

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-4 shrink-0 shadow-xs flex items-center justify-between">
            <div className="flex items-center gap-3 min-w-0">
              <div className="w-11 h-11 rounded-2xl bg-[#FAF8F2] border-2 border-zinc-900 flex items-center justify-center font-extrabold text-sm text-zinc-900 shadow-xs shrink-0">
                {user?.name ? user.name.slice(0, 2).toUpperCase() : 'КП'}
              </div>
              <div className="min-w-0">
                <h1 className="text-sm font-extrabold text-zinc-900 truncate">{user?.name || 'Покупатель'}</h1>
                <p className="text-[11px] text-zinc-500 font-medium truncate">{user?.email}</p>
                {user?.budget_max && (
                  <p className="text-[11px] text-zinc-900 font-bold truncate">
                    Бюджет: до {formatPrice(user.budget_max)}
                  </p>
                )}
              </div>
            </div>
          </div>

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-4 flex flex-col flex-1 min-h-0 shadow-xs space-y-3">
            <div className="flex items-center justify-between border-b border-zinc-300 pb-2 shrink-0">
              <h2 className="text-xs font-extrabold text-zinc-900 uppercase tracking-wider">Мои обращения и заявки</h2>
              <button
                onClick={handleCreateGeneralChat}
                disabled={creatingGeneralChat}
                className="px-2.5 py-1 bg-white hover:bg-[#FAF8F2] text-zinc-900 text-[11px] font-bold rounded-xl border border-zinc-900 flex items-center gap-1 transition-colors cursor-pointer disabled:opacity-50"
                title="Создать новое обращение к менеджеру по общим вопросам"
              >
                <Plus className="w-3 h-3" />
                <span>Общий вопрос</span>
              </button>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto space-y-2.5 pr-1">
              {loading ? (
                <div className="py-8 text-center text-zinc-400">
                  <RefreshCw className="w-5 h-5 animate-spin mx-auto text-zinc-900" />
                </div>
              ) : filteredSessions.length === 0 ? (
                <div className="py-6 text-center text-zinc-500 text-xs font-medium">
                  У вас пока нет активных чатов. Выберите квартиру в каталоге и напишите менеджеру!
                </div>
              ) : (
                filteredSessions.map((sess) => {
                  const apt = sess.id_apartment ? apartmentsMap[sess.id_apartment] : null;
                  const isActive = activeSessionId === sess.id;
                  const badge = getChatSessionStatusBadge(sess.status);

                  return (
                    <div
                      key={sess.id}
                      onClick={() => setActiveSessionId(sess.id)}
                      className={`p-3 rounded-2xl border-2 transition-all cursor-pointer relative group ${
                        isActive
                          ? 'bg-[#FAF8F2] border-zinc-900 shadow-xs'
                          : 'bg-white border-zinc-300 hover:border-zinc-900'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-1 mb-1">
                        <span className="text-xs font-extrabold text-zinc-900 truncate">
                          {apt ? `Квартира №${apt.number}` : 'Общий диалог'}
                        </span>

                        <div className="flex items-center gap-1">
                          <span className="text-[10px] text-zinc-400 font-semibold shrink-0">
                            {formatDate(sess.updated_at)}
                          </span>
                          <button
                            type="button"
                            onClick={(e) => handleDeleteSession(e, sess.id)}
                            className="p-1 rounded-lg text-zinc-400 hover:text-rose-600 hover:bg-rose-50 transition-colors ml-1"
                            title="Удалить диалог из списка"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>

                      {apt ? (
                        <p className="text-[11px] text-zinc-600 font-medium truncate">
                          {apt.rooms}-комн., {apt.area} м² • {formatPrice(apt.price)}
                        </p>
                      ) : (
                        <p className="text-[11px] text-zinc-400">Подбор и консультация</p>
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
                        <span className="text-[11px] text-zinc-900 font-bold">
                          Открыть диалог →
                        </span>
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
              onSessionUpdated={loadData}
            />
          ) : (
            <div className="h-full flex flex-col items-center justify-center p-8 bg-white rounded-3xl border-2 border-zinc-900 text-center text-zinc-400 space-y-1 shadow-xs">
              <h3 className="text-sm font-extrabold text-zinc-900">Выберите диалог</h3>
              <p className="text-xs text-zinc-500 max-w-sm">
                Выберите диалог из списка слева или задайте вопрос по понравившейся квартире из каталога.
              </p>
            </div>
          )}
        </div>

      </div>

    </div>
  );
};

