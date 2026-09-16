import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { ShieldCheck, TrendingUp, Users, FileText, CheckCircle2, AlertCircle, RefreshCw, SlidersHorizontal } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { dealsApi } from '../api/deals';
import { chatsApi } from '../api/chats';
import { staffApi } from '../api/discounts';
import { Deal, Offer, ChatSession, User } from '../types';
import { formatPrice, getDealStatusBadge, formatDate } from '../lib/utils';

export const SupervisorDashboardPage: React.FC = () => {
  const { user } = useAuth();

  const [deals, setDeals] = useState<Deal[]>([]);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [staffUsers, setStaffUsers] = useState<User[]>([]);

  const [activeTab, setActiveTab] = useState<'deals' | 'offers' | 'staff'>('offers');
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [rejectReason, setRejectReason] = useState<{ id: number; text: string } | null>(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const [dealsData, offersData, sessionsData, usersData] = await Promise.all([
        dealsApi.getAllDeals().catch(() => []),
        dealsApi.getAllOffers().catch(() => []),
        chatsApi.getAllSessions().catch(() => []),
        staffApi.getUsers().catch(() => []),
      ]);

      setDeals(dealsData);
      setOffers(offersData);
      setSessions(sessionsData);
      setStaffUsers(usersData);
    } catch (err) {
      console.error('Failed to load supervisor data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleApproveOffer = async (offerId: number) => {
    setActionLoading(offerId);
    try {
      await dealsApi.approveOffer(offerId);
      await loadData();
    } catch (err: any) {
      alert(err.message || 'Ошибка согласования');
    } finally {
      setActionLoading(null);
    }
  };

  const handleRejectOffer = async (offerId: number, reason: string) => {
    setActionLoading(offerId);
    try {
      await dealsApi.rejectOffer(offerId, reason || 'Отклонено руководителем');
      setRejectReason(null);
      await loadData();
    } catch (err: any) {
      alert(err.message || 'Ошибка отклонения');
    } finally {
      setActionLoading(null);
    }
  };

  const totalVolume = deals
    .filter((d) => d.status !== 'cancelled')
    .reduce((sum, d) => sum + d.total_price, 0);

  const pendingOffers = offers.filter((o) => o.status === 'pending_approval');
  const completedDeals = deals.filter((d) => d.status === 'completed');

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">

      <div className="bg-white rounded-3xl border-2 border-zinc-900 p-6 flex flex-col md:flex-row items-start md:items-center justify-between gap-6 shadow-xs">
        <div className="flex items-center gap-4">
          <div className="w-14 h-14 rounded-2xl bg-[#FAF8F2] border-2 border-zinc-900 flex items-center justify-center text-zinc-900 shadow-xs">
            <ShieldCheck className="w-7 h-7" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-extrabold text-zinc-900">{user?.name || 'Руководитель отдела'}</h1>
            </div>
            <p className="text-xs text-zinc-500 font-medium mt-0.5">{user?.email}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Link
            to="/supervisor/discounts"
            className="px-5 py-2.5 bg-zinc-900 hover:bg-zinc-800 text-white rounded-xl text-xs font-bold flex items-center gap-2 border-2 border-zinc-900 transition-all cursor-pointer shadow-xs"
          >
            <SlidersHorizontal className="w-4 h-4" />
            Матрица скидок
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white p-5 rounded-2xl border-2 border-zinc-900 shadow-xs">
          <div className="flex items-center justify-between text-zinc-500 mb-1">
            <span className="text-xs font-bold uppercase tracking-wider">Общий объём продаж</span>
            <TrendingUp className="w-4 h-4 text-zinc-900" />
          </div>
          <p className="text-2xl font-extrabold text-zinc-900">{formatPrice(totalVolume)}</p>
          <p className="text-[11px] text-zinc-500 font-semibold mt-1">{deals.length} сделок в системе</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border-2 border-zinc-900 shadow-xs">
          <div className="flex items-center justify-between text-zinc-500 mb-1">
            <span className="text-xs font-bold uppercase tracking-wider">На согласовании</span>
            <AlertCircle className="w-4 h-4 text-amber-500" />
          </div>
          <p className="text-2xl font-extrabold text-amber-600">{pendingOffers.length}</p>
          <p className="text-[11px] text-zinc-500 font-semibold mt-1">Требуют вашего решения</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border-2 border-zinc-900 shadow-xs">
          <div className="flex items-center justify-between text-zinc-500 mb-1">
            <span className="text-xs font-bold uppercase tracking-wider">Завершённые сделки</span>
            <CheckCircle2 className="w-4 h-4 text-emerald-600" />
          </div>
          <p className="text-2xl font-extrabold text-emerald-700">{completedDeals.length}</p>
          <p className="text-[11px] text-zinc-500 font-semibold mt-1">Договоры оплачены</p>
        </div>

        <div className="bg-white p-5 rounded-2xl border-2 border-zinc-900 shadow-xs">
          <div className="flex items-center justify-between text-zinc-500 mb-1">
            <span className="text-xs font-bold uppercase tracking-wider">Сотрудники отдела</span>
            <Users className="w-4 h-4 text-zinc-900" />
          </div>
          <p className="text-2xl font-extrabold text-zinc-900">
            {staffUsers.filter((u) => u.role === 'manager' || u.role === 'supervisor').length}
          </p>
          <p className="text-[11px] text-zinc-500 font-semibold mt-1">Менеджеров в отделе</p>
        </div>
      </div>

      <div className="bg-white rounded-3xl border-2 border-zinc-900 p-5 sm:p-6 space-y-5 shadow-xs">

        <div className="flex flex-wrap items-center gap-2 border-b border-zinc-300 pb-3">
          <button
            onClick={() => setActiveTab('offers')}
            className={`px-4 py-2 rounded-xl text-xs font-bold transition-all border-2 cursor-pointer flex items-center gap-2 ${
              activeTab === 'offers'
                ? 'bg-zinc-900 text-white border-zinc-900 shadow-xs'
                : 'bg-white border-zinc-900 text-zinc-900 hover:bg-[#FAF8F2]'
            }`}
          >
            <FileText className="w-4 h-4" />
            Согласование КП и скидок ({pendingOffers.length})
          </button>

          <button
            onClick={() => setActiveTab('deals')}
            className={`px-4 py-2 rounded-xl text-xs font-bold transition-all border-2 cursor-pointer flex items-center gap-2 ${
              activeTab === 'deals'
                ? 'bg-zinc-900 text-white border-zinc-900 shadow-xs'
                : 'bg-white border-zinc-900 text-zinc-900 hover:bg-[#FAF8F2]'
            }`}
          >
            <TrendingUp className="w-4 h-4" />
            Все сделки отдела ({deals.length})
          </button>

          <button
            onClick={() => setActiveTab('staff')}
            className={`px-4 py-2 rounded-xl text-xs font-bold transition-all border-2 cursor-pointer flex items-center gap-2 ${
              activeTab === 'staff'
                ? 'bg-zinc-900 text-white border-zinc-900 shadow-xs'
                : 'bg-white border-zinc-900 text-zinc-900 hover:bg-[#FAF8F2]'
            }`}
          >
            <Users className="w-4 h-4" />
            Команда и менеджеры
          </button>
        </div>

        {activeTab === 'offers' && (
          <div className="space-y-4">
            <h3 className="text-xs font-extrabold text-zinc-900 uppercase tracking-wider">
              Коммерческие предложения, ожидающие утверждения
            </h3>

            {pendingOffers.length === 0 ? (
              <div className="py-8 text-center text-zinc-500 bg-[#FAF8F2] rounded-2xl border-2 border-zinc-900">
                <CheckCircle2 className="w-6 h-6 mx-auto text-emerald-600 mb-1" />
                <p className="text-xs font-bold">Нет предложений, требующих утверждения</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {pendingOffers.map((offer) => (
                  <div
                    key={offer.id}
                    className="p-4 sm:p-5 bg-[#FFFDF8] rounded-2xl border-2 border-zinc-900 space-y-3 shadow-xs"
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-extrabold text-zinc-900 text-sm">
                        КП #{offer.id} (версия {offer.version})
                      </span>
                      <span className="px-2.5 py-0.5 rounded-full text-xs font-extrabold bg-[#FEF7EE] text-amber-950 border border-zinc-900">
                        Скидка {offer.discount_percent}%
                      </span>
                    </div>

                    <p className="text-xs text-zinc-800 leading-relaxed bg-white p-3.5 rounded-xl border border-zinc-300 font-medium">
                      {offer.generated_text || 'Запрошена скидка выше стандартного лимита менеджера.'}
                    </p>

                    <div className="flex justify-between text-xs text-zinc-800 font-bold pt-1">
                      <span>Итоговая стоимость:</span>
                      <strong className="text-zinc-900 text-sm font-extrabold">{formatPrice(offer.final_price)}</strong>
                    </div>

                    <div className="flex items-center justify-end gap-2 pt-2 border-t border-zinc-200">
                      {rejectReason?.id === offer.id ? (
                        <div className="flex items-center gap-2 w-full">
                          <input
                            type="text"
                            placeholder="Причина отклонения..."
                            value={rejectReason.text}
                            onChange={(e) =>
                              setRejectReason({ id: offer.id, text: e.target.value })
                            }
                            className="flex-1 px-3 py-1.5 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900"
                          />
                          <button
                            onClick={() => handleRejectOffer(offer.id, rejectReason.text)}
                            className="px-3.5 py-1.5 bg-rose-600 hover:bg-rose-700 text-white rounded-xl text-xs font-bold border-2 border-zinc-900 cursor-pointer"
                          >
                            Отклонить
                          </button>
                        </div>
                      ) : (
                        <>
                          <button
                            onClick={() => setRejectReason({ id: offer.id, text: '' })}
                            className="px-3.5 py-1.5 rounded-xl text-xs font-bold text-zinc-900 bg-white hover:bg-rose-50 border-2 border-zinc-900 transition-colors cursor-pointer"
                          >
                            Отклонить
                          </button>
                          <button
                            onClick={() => handleApproveOffer(offer.id)}
                            disabled={actionLoading === offer.id}
                            className="px-4 py-1.5 rounded-xl text-xs font-bold text-white bg-emerald-600 hover:bg-emerald-700 border-2 border-zinc-900 transition-colors flex items-center gap-1.5 disabled:opacity-50 cursor-pointer"
                          >
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            Согласовать скидку
                          </button>
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'deals' && (
          <div className="space-y-4">
            <h3 className="text-xs font-extrabold text-zinc-900 uppercase tracking-wider">Реестр сделок отдела</h3>

            <div className="overflow-x-auto rounded-2xl border-2 border-zinc-900">
              <table className="w-full text-left text-xs border-collapse bg-white">
                <thead>
                  <tr className="border-b-2 border-zinc-900 bg-[#FAF8F2] text-zinc-900 font-extrabold uppercase tracking-wider text-[10px]">
                    <th className="py-3 px-4">№ Сделки</th>
                    <th className="py-3 px-4">Клиент</th>
                    <th className="py-3 px-4">Менеджер</th>
                    <th className="py-3 px-4">Квартира</th>
                    <th className="py-3 px-4">Скидка</th>
                    <th className="py-3 px-4">Итог</th>
                    <th className="py-3 px-4">Статус</th>
                    <th className="py-3 px-4">Дата</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-200 font-semibold text-zinc-800">
                  {deals.map((deal) => {
                    const badge = getDealStatusBadge(deal.status);
                    return (
                      <tr key={deal.id} className="hover:bg-[#FAF8F2]/60 transition-colors">
                        <td className="py-3 px-4 font-extrabold text-zinc-900">#{deal.id}</td>
                        <td className="py-3 px-4">Клиент #{deal.id_user}</td>
                        <td className="py-3 px-4">Сотрудник #{deal.id_employee}</td>
                        <td className="py-3 px-4">Кв. #{deal.id_apartment}</td>
                        <td className="py-3 px-4 text-emerald-800 font-extrabold">{deal.percent_discount}%</td>
                        <td className="py-3 px-4 font-extrabold text-zinc-900">{formatPrice(deal.total_price)}</td>
                        <td className="py-3 px-4">
                          <span className={`px-2.5 py-0.5 rounded-full text-[10px] font-extrabold border ${badge.color}`}>
                            {badge.label}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-zinc-500">{formatDate(deal.created_at)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === 'staff' && (
          <div className="space-y-4">
            <h3 className="text-xs font-extrabold text-zinc-900 uppercase tracking-wider">Сотрудники отдела продаж</h3>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {staffUsers.map((u) => (
                <div key={u.id} className="p-4 bg-[#FAF8F2] rounded-2xl border-2 border-zinc-900 flex items-center gap-3.5 shadow-xs">
                  <div className="w-10 h-10 rounded-xl bg-white border-2 border-zinc-900 flex items-center justify-center font-extrabold text-xs text-zinc-900">
                    {u.name ? u.name.slice(0, 2).toUpperCase() : 'С'}
                  </div>
                  <div>
                    <h4 className="font-extrabold text-zinc-900 text-xs">{u.name || u.email}</h4>
                    <p className="text-[11px] text-zinc-500 font-medium">{u.email}</p>
                    <span className="text-[10px] text-zinc-800 font-extrabold capitalize mt-0.5 block">
                      {u.role === 'supervisor' ? 'Руководитель' : u.role === 'manager' ? 'Менеджер' : 'Пользователь'}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

      </div>

    </div>
  );
};
