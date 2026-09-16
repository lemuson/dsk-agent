import React, { useState, useEffect } from 'react';
import { useParams, useSearchParams, useNavigate, Link } from 'react-router-dom';
import {
  FileText, Download, CheckCircle2, XCircle, ArrowLeft,
  Building2, User as UserIcon, ShieldAlert, Sparkles, RefreshCw, Check, Eye
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { dealsApi, CalculateOfferResult } from '../api/deals';
import { apartmentsApi } from '../api/apartments';
import { chatsApi } from '../api/chats';
import { Deal, Offer, Apartment, ChatSession } from '../types';
import { formatPrice, formatDate } from '../lib/utils';

interface AncillaryUnit {
  id: number;
  building_id: number;
  kind: 'parking' | 'storage';
  number: string;
  area: number;
  price: number;
  status: string;
}

export const DealDetailsPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { user } = useAuth();

  const isStaff = user?.role === 'manager' || user?.role === 'supervisor';
  const isSupervisor = user?.role === 'supervisor';

  const sessionIdParam = searchParams.get('session');
  const aptIdParam = searchParams.get('apartment') || searchParams.get('apartment_id');
  const userIdParam = searchParams.get('user') || searchParams.get('user_id');

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [deal, setDeal] = useState<Deal | null>(null);
  const [apartment, setApartment] = useState<Apartment | null>(null);
  const [session, setSession] = useState<ChatSession | null>(null);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [latestOffer, setLatestOffer] = useState<Offer | null>(null);

  const [ancillaryUnits, setAncillaryUnits] = useState<AncillaryUnit[]>([]);
  const [selectedParkingId, setSelectedParkingId] = useState<number | undefined>(undefined);
  const [selectedStorageId, setSelectedStorageId] = useState<number | undefined>(undefined);
  const [discountPercent, setDiscountPercent] = useState<number>(0);
  const [customText, setCustomText] = useState<string>('');

  const [calculation, setCalculation] = useState<CalculateOfferResult | null>(null);
  const [calcLoading, setCalcLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const [pdfBlobUrl, setPdfBlobUrl] = useState<string | null>(null);
  const [loadingPdf, setLoadingPdf] = useState(false);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);

      let currentDeal: Deal | null = null;
      let aptId: number | null = null;

      if (id && id !== 'new') {
        const dealIdNum = parseInt(id, 10);
        currentDeal = await dealsApi.getDeal(dealIdNum, isStaff);
        setDeal(currentDeal);
        aptId = currentDeal.id_apartment;
        setDiscountPercent(currentDeal.percent_discount || 0);

        try {
          const allOffers = isStaff ? await dealsApi.getAllOffers() : await dealsApi.getMyOffers();
          const dealOffers = allOffers.filter((o) => o.deal_id === dealIdNum);
          setOffers(dealOffers);
          if (dealOffers.length > 0) {
            const latest = dealOffers[0];
            setLatestOffer(latest);
            setSelectedParkingId(latest.parking_unit_id);
            setSelectedStorageId(latest.storage_unit_id);
            setDiscountPercent(parseFloat(latest.discount_percent) || currentDeal.percent_discount || 0);
            setCustomText(latest.generated_text || '');
          }
        } catch (e) {
          console.warn('Failed to load offers:', e);
        }
      } else {
        if (aptIdParam) aptId = parseInt(aptIdParam, 10);
        if (sessionIdParam) {
          try {
            const sess = await chatsApi.getSession(parseInt(sessionIdParam, 10), isStaff);
            setSession(sess);
            if (!aptId && sess.id_apartment) aptId = sess.id_apartment;
          } catch (e) {
            console.warn('Failed to load session:', e);
          }
        }
      }

      if (aptId) {
        try {
          const aptData = await apartmentsApi.getApartment(aptId);
          setApartment(aptData);
          if (aptData.building_id) {
            const units = await apartmentsApi.getAncillaryUnits(aptData.building_id);
            setAncillaryUnits(units);
          }
        } catch (e) {
          console.warn('Failed to load apartment:', e);
        }
      }
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки сделки');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [id, isStaff]);

  useEffect(() => {
    if (!deal && !apartment) return;
    if (deal) {
      recalculateOffer(deal.id, discountPercent, selectedParkingId, selectedStorageId);
    }
  }, [deal, discountPercent, selectedParkingId, selectedStorageId]);

  useEffect(() => {
    let activeUrl: string | null = null;
    const fetchPdfPreview = async () => {
      if (!latestOffer || latestOffer.status !== 'approved') {
        setPdfBlobUrl(null);
        return;
      }
      try {
        setLoadingPdf(true);
        const blob = await dealsApi.downloadOfferPdf(latestOffer.id, isStaff);
        activeUrl = window.URL.createObjectURL(blob);
        setPdfBlobUrl(activeUrl);
      } catch (e) {
        console.warn('Failed to load PDF preview:', e);
      } finally {
        setLoadingPdf(false);
      }
    };

    fetchPdfPreview();

    return () => {
      if (activeUrl) window.URL.revokeObjectURL(activeUrl);
    };
  }, [latestOffer, isStaff]);

  const recalculateOffer = async (
    dealId: number,
    discount: number,
    parkingId?: number,
    storageId?: number
  ) => {
    if (!isStaff) return;
    try {
      setCalcLoading(true);
      const res = await dealsApi.calculateOffer({
        deal_id: dealId,
        discount_percent: discount.toFixed(2),
        parking_unit_id: parkingId,
        storage_unit_id: storageId,
      });
      setCalculation(res);
    } catch (err) {
      console.warn('Calculate offer error:', err);
    } finally {
      setCalcLoading(false);
    }
  };

  const handleCreateDealAndOffer = async () => {
    try {
      setActionLoading(true);
      setError(null);

      let targetDeal = deal;
      const clientUserId = userIdParam ? parseInt(userIdParam, 10) : session?.id_user || user?.id;
      const targetAptId = apartment?.id || (aptIdParam ? parseInt(aptIdParam, 10) : 0);
      const chatSessId = sessionIdParam ? parseInt(sessionIdParam, 10) : session?.id;

      if (!targetDeal) {
        if (!clientUserId || !targetAptId) {
          throw new Error('Недостаточно данных для создания сделки (не указан клиент или квартира)');
        }

        targetDeal = await dealsApi.createDeal({
          id_user: clientUserId,
          id_apartment: targetAptId,
          id_chat_session: chatSessId,
          percent_discount: discountPercent,
        });
        setDeal(targetDeal);
      }

      const offerText = customText || `Коммерческое предложение по квартире №${apartment?.number || ''}. Скидка: ${discountPercent}%.`;
      const createdOffer = await dealsApi.createOffer({
        request_id: `req-${Date.now()}`,
        deal_id: targetDeal.id,
        discount_percent: discountPercent.toFixed(2),
        generated_text: offerText,
        parking_unit_id: selectedParkingId,
        storage_unit_id: selectedStorageId,
      });

      setLatestOffer(createdOffer);

      setSuccessMessage('Коммерческое предложение сформировано и направлено на согласование!');
      navigate(`/deals/${targetDeal.id}`, { replace: true });
      loadData();
    } catch (err: any) {
      setError(err.message || 'Ошибка оформления');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDownloadPdf = async () => {
    if (!latestOffer) return;
    try {
      setActionLoading(true);
      const blob = await dealsApi.downloadOfferPdf(latestOffer.id, isStaff);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `offer-deal-${deal?.id || 'dsk'}-v${latestOffer.version}.pdf`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err: any) {
      setError(err.message || 'Ошибка скачивания PDF');
    } finally {
      setActionLoading(false);
    }
  };

  const handleApprove = async () => {
    if (!latestOffer) return;
    try {
      setActionLoading(true);
      if (isSupervisor) {
        await dealsApi.approveOffer(latestOffer.id);
      } else {
        await dealsApi.updateDealStatus(deal!.id, 'contract', discountPercent, false);
      }

      setSuccessMessage('Предложение успешно согласовано и подтверждено!');
      loadData();
    } catch (err: any) {
      setError(err.message || 'Ошибка согласования');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async () => {
    if (!latestOffer) return;
    const reason = prompt('Укажите причину отклонения:') || 'Отклонено пользователем/руководителем';
    try {
      setActionLoading(true);
      if (isSupervisor) {
        await dealsApi.rejectOffer(latestOffer.id, reason);
      } else {
        await dealsApi.updateDealStatus(deal!.id, 'pending', discountPercent, false);
      }

      setSuccessMessage('Предложение отклонено. Заявка осталась в работе.');
      loadData();
    } catch (err: any) {
      setError(err.message || 'Ошибка');
    } finally {
      setActionLoading(false);
    }
  };

  const handleCloseDeal = async () => {
    if (!deal) return;
    if (!window.confirm(`Вы уверены, что хотите закрыть сделку #${deal.id}? Статус заявки и сделки будет переведен в «Завершено», а квартира — в «Продано».`)) {
      return;
    }
    try {
      setActionLoading(true);
      await dealsApi.updateDealStatus(deal.id, 'completed', discountPercent, isStaff);
      setSuccessMessage('Сделка успешно закрыта! Статус: Завершено, квартира: Продано.');
      loadData();
    } catch (err: any) {
      setError(err.message || 'Ошибка закрытия сделки');
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="max-w-5xl mx-auto px-4 py-16 text-center">
        <RefreshCw className="w-8 h-8 animate-spin mx-auto text-zinc-900 mb-3" />
        <p className="text-sm font-bold text-zinc-600">Загрузка информации по сделке...</p>
      </div>
    );
  }

  const basePrice = apartment?.price || deal?.base_price || 0;
  const currentParking = ancillaryUnits.find((u) => u.id === selectedParkingId);
  const currentStorage = ancillaryUnits.find((u) => u.id === selectedStorageId);
  const parkingPrice = currentParking?.price || 0;
  const storagePrice = currentStorage?.price || 0;
  const rawTotal = basePrice + parkingPrice + storagePrice;
  const discountVal = (rawTotal * discountPercent) / 100;
  const finalPrice = calculation ? calculation.final_price : rawTotal - discountVal;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">

      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate(-1)}
            className="w-10 h-10 rounded-2xl bg-white border-2 border-zinc-900 flex items-center justify-center text-zinc-700 hover:text-zinc-900 hover:bg-[#FAF8F2] shadow-xs cursor-pointer"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl sm:text-2xl font-black text-zinc-900 tracking-tight">
                {deal ? `Сделка #${deal.id}` : 'Оформление сделки и коммерческого предложения'}
              </h1>
              {deal && (
                <span className={`px-2.5 py-0.5 rounded-full text-xs font-black border-2 border-zinc-900 ${
                  deal.status === 'contract'
                    ? 'bg-emerald-100 text-emerald-950'
                    : 'bg-[#FAF8F2] text-zinc-900'
                }`}>
                  {deal.status === 'pending' && (latestOffer ? 'На согласовании' : 'В работе')}
                  {deal.status === 'contract' && 'Согласовано (Договор)'}
                  {deal.status === 'completed' && 'Завершена'}
                  {deal.status === 'cancelled' && 'Отменена'}
                </span>
              )}
            </div>
            <p className="text-xs text-zinc-500 font-semibold mt-0.5">
              {deal ? `Создана ${formatDate(deal.created_at)}` : 'Заполните параметры и сформируйте коммерческое предложение'}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2.5 flex-wrap">
          {latestOffer && latestOffer.status === 'approved' && (
            <button
              onClick={handleDownloadPdf}
              disabled={actionLoading}
              className="inline-flex items-center gap-2 px-4 py-2.5 bg-white border-2 border-zinc-900 rounded-2xl text-xs font-black text-zinc-900 hover:bg-[#FAF8F2] shadow-xs transition-all cursor-pointer"
            >
              <Download className="w-4 h-4" />
              Скачать PDF
            </button>
          )}

          {isStaff && (
            <button
              onClick={handleCreateDealAndOffer}
              disabled={actionLoading}
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-zinc-900 border-2 border-zinc-900 rounded-2xl text-xs font-black text-white hover:bg-zinc-800 shadow-xs transition-all cursor-pointer"
            >
              <FileText className="w-4 h-4" />
              {latestOffer ? 'Обновить предложение' : 'Сформировать сделку'}
            </button>
          )}

          {isStaff && deal && deal.status === 'contract' && (
            <button
              onClick={handleCloseDeal}
              disabled={actionLoading}
              className="inline-flex items-center gap-2 px-4 py-2.5 bg-emerald-600 border-2 border-zinc-900 rounded-2xl text-xs font-black text-white hover:bg-emerald-700 shadow-xs transition-all cursor-pointer"
              title="Закрыть сделку и перевести квартиру в статус «Продано»"
            >
              <CheckCircle2 className="w-4 h-4" />
              Закрыть сделку
            </button>
          )}

          {isSupervisor && latestOffer && latestOffer.status === 'pending_approval' && (
            <>
              <button
                onClick={handleApprove}
                disabled={actionLoading}
                className="inline-flex items-center gap-2 px-4 py-2.5 bg-emerald-600 border-2 border-emerald-950 rounded-2xl text-xs font-black text-white hover:bg-emerald-700 shadow-xs transition-all cursor-pointer"
              >
                <CheckCircle2 className="w-4 h-4" />
                Согласовать скидку
              </button>
              <button
                onClick={handleReject}
                disabled={actionLoading}
                className="inline-flex items-center gap-2 px-4 py-2.5 bg-white border-2 border-rose-600 rounded-2xl text-xs font-black text-rose-700 hover:bg-rose-50 shadow-xs transition-all cursor-pointer"
              >
                <XCircle className="w-4 h-4" />
                Отклонить
              </button>
            </>
          )}

          {!isStaff && deal && deal.status === 'pending' && (
            <>
              <button
                onClick={handleApprove}
                disabled={actionLoading}
                className="inline-flex items-center gap-2 px-4 py-2.5 bg-emerald-600 border-2 border-emerald-950 rounded-2xl text-xs font-black text-white hover:bg-emerald-700 shadow-xs transition-all cursor-pointer"
              >
                <CheckCircle2 className="w-4 h-4" />
                Подтвердить сделку
              </button>
              <button
                onClick={handleReject}
                disabled={actionLoading}
                className="inline-flex items-center gap-2 px-4 py-2.5 bg-white border-2 border-rose-600 rounded-2xl text-xs font-black text-rose-700 hover:bg-rose-50 shadow-xs transition-all cursor-pointer"
              >
                <XCircle className="w-4 h-4" />
                Отклонить предложение
              </button>
            </>
          )}
        </div>
      </div>

      {error && (
        <div className="p-4 rounded-2xl bg-rose-50 border-2 border-rose-900 text-rose-900 text-xs font-bold shadow-xs">
          {error}
        </div>
      )}
      {successMessage && (
        <div className="p-4 rounded-2xl bg-emerald-50 border-2 border-emerald-900 text-emerald-900 text-xs font-bold shadow-xs flex items-center gap-2">
          <Check className="w-4 h-4 shrink-0" />
          {successMessage}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">

        <div className="lg:col-span-6 space-y-6">

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-5 shadow-xs space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-200 pb-3">
              <div className="flex items-center gap-2">
                <Building2 className="w-5 h-5 text-zinc-900" />
                <h2 className="text-base font-black text-zinc-900">Данные по объекту</h2>
              </div>
              {apartment && (
                <span className="text-xs font-black px-2.5 py-1 bg-[#FAF8F2] border border-zinc-300 rounded-xl text-zinc-800">
                  №{apartment.number}
                </span>
              )}
            </div>

            {apartment ? (
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 pt-1">
                <div>
                  <p className="text-[11px] font-bold text-zinc-400">Комнатность</p>
                  <p className="text-sm font-black text-zinc-900">{apartment.rooms}-комнатная</p>
                </div>
                <div>
                  <p className="text-[11px] font-bold text-zinc-400">Площадь</p>
                  <p className="text-sm font-black text-zinc-900">{apartment.area} м²</p>
                </div>
                <div>
                  <p className="text-[11px] font-bold text-zinc-400">Этаж</p>
                  <p className="text-sm font-black text-zinc-900">{apartment.floor} этаж</p>
                </div>
                <div>
                  <p className="text-[11px] font-bold text-zinc-400">Базовая стоимость</p>
                  <p className="text-sm font-black text-zinc-900">{formatPrice(apartment.price)}</p>
                </div>
              </div>
            ) : (
              <p className="text-xs text-zinc-500">Квартира не выбрана или не найдена.</p>
            )}
          </div>

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-5 shadow-xs space-y-5">
            <div className="flex items-center justify-between border-b border-zinc-200 pb-3">
              <h2 className="text-base font-black text-zinc-900">Конфигурация опций</h2>
              <span className="text-xs text-zinc-400 font-bold">Паркинг, кладовые и скидка</span>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">

              <div className="space-y-2">
                <label className="text-xs font-bold text-zinc-700">Машиноместо в паркинге</label>
                <select
                  disabled={!isStaff}
                  value={selectedParkingId || ''}
                  onChange={(e) => setSelectedParkingId(e.target.value ? parseInt(e.target.value, 10) : undefined)}
                  className="w-full px-3.5 py-2.5 bg-[#FAF8F2] border-2 border-zinc-900 rounded-2xl text-xs font-bold text-zinc-900 focus:outline-none focus:bg-white cursor-pointer disabled:opacity-75"
                >
                  <option value="">Без паркинга</option>
                  {ancillaryUnits
                    .filter((u) => u.kind === 'parking')
                    .map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.number} ({p.area} м²) — {formatPrice(p.price)}
                      </option>
                    ))}
                </select>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-bold text-zinc-700">Индивидуальная кладовая</label>
                <select
                  disabled={!isStaff}
                  value={selectedStorageId || ''}
                  onChange={(e) => setSelectedStorageId(e.target.value ? parseInt(e.target.value, 10) : undefined)}
                  className="w-full px-3.5 py-2.5 bg-[#FAF8F2] border-2 border-zinc-900 rounded-2xl text-xs font-bold text-zinc-900 focus:outline-none focus:bg-white cursor-pointer disabled:opacity-75"
                >
                  <option value="">Без кладовой</option>
                  {ancillaryUnits
                    .filter((u) => u.kind === 'storage')
                    .map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.number} ({s.area} м²) — {formatPrice(s.price)}
                      </option>
                    ))}
                </select>
              </div>
            </div>

            <div className="pt-2 border-t border-zinc-100 space-y-3">
              <div className="flex items-center justify-between">
                <label className="text-xs font-bold text-zinc-700">
                  Персональная скидка: <span className="font-black text-zinc-900">{discountPercent}%</span>
                </label>
                {calculation?.requires_approval && (
                  <span className="inline-flex items-center gap-1 text-[11px] font-bold text-amber-700 bg-amber-50 border border-amber-300 px-2 py-0.5 rounded-full">
                    <ShieldAlert className="w-3.5 h-3.5" />
                    Требует согласования руководителя
                  </span>
                )}
              </div>
              {isStaff ? (
                <div className="flex items-center gap-4">
                  <input
                    type="range"
                    min="0"
                    max="15"
                    step="0.5"
                    value={discountPercent}
                    onChange={(e) => setDiscountPercent(parseFloat(e.target.value))}
                    className="flex-1 accent-zinc-900 cursor-pointer h-2 bg-zinc-200 rounded-lg"
                  />
                  <input
                    type="number"
                    min="0"
                    max="15"
                    step="0.1"
                    value={discountPercent}
                    onChange={(e) => setDiscountPercent(parseFloat(e.target.value) || 0)}
                    className="w-16 px-2 py-1 bg-[#FAF8F2] border-2 border-zinc-900 rounded-xl text-xs font-black text-center focus:outline-none"
                  />
                </div>
              ) : (
                <p className="text-xs font-bold text-zinc-500">Скидка зафиксирована в коммерческом предложении.</p>
              )}
            </div>

            {isStaff && (
              <div className="space-y-2 pt-2">
                <label className="text-xs font-bold text-zinc-700">Примечание к коммерческому предложению</label>
                <textarea
                  rows={3}
                  value={customText}
                  onChange={(e) => setCustomText(e.target.value)}
                  placeholder="Дополнительные условия, спецпредложения или комментарии для клиента..."
                  className="w-full px-3.5 py-2.5 bg-[#FAF8F2] border-2 border-zinc-900 rounded-2xl text-xs font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-white resize-none"
                />
              </div>
            )}
          </div>

        </div>

        <div className="lg:col-span-6 space-y-6">

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-5 shadow-xs space-y-4">
            <h2 className="text-base font-black text-zinc-900 border-b border-zinc-200 pb-3">Расчет стоимости сделки</h2>

            <div className="space-y-2.5 text-xs">
              <div className="flex items-center justify-between text-zinc-600">
                <span>Квартира №{apartment?.number || ''}:</span>
                <span className="font-bold text-zinc-900">{formatPrice(basePrice)}</span>
              </div>

              {currentParking && (
                <div className="flex items-center justify-between text-zinc-600">
                  <span>Машиноместо ({currentParking.number}):</span>
                  <span className="font-bold text-zinc-900">+{formatPrice(parkingPrice)}</span>
                </div>
              )}

              {currentStorage && (
                <div className="flex items-center justify-between text-zinc-600">
                  <span>Кладовая ({currentStorage.number}):</span>
                  <span className="font-bold text-zinc-900">+{formatPrice(storagePrice)}</span>
                </div>
              )}

              {discountPercent > 0 && (
                <div className="flex items-center justify-between text-emerald-700 font-bold pt-1 border-t border-zinc-100">
                  <span>Скидка ({discountPercent}%):</span>
                  <span>-{formatPrice(calculation ? calculation.discount_amount : discountVal)}</span>
                </div>
              )}
            </div>

            <div className="pt-4 border-t-2 border-zinc-900 flex items-center justify-between">
              <div>
                <span className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block">Финальная стоимость</span>
                <span className="text-2xl font-black text-zinc-900">{formatPrice(finalPrice)}</span>
              </div>

              {latestOffer && latestOffer.status === 'approved' && (
                <button
                  onClick={handleDownloadPdf}
                  className="px-4 py-2.5 bg-[#FAF8F2] border-2 border-zinc-900 rounded-2xl text-xs font-black text-zinc-900 hover:bg-zinc-900 hover:text-white transition-all flex items-center gap-1.5 shadow-xs cursor-pointer"
                >
                  <Download className="w-4 h-4" />
                  Скачать PDF
                </button>
              )}
            </div>
          </div>

          <div className="bg-white rounded-3xl border-2 border-zinc-900 p-5 shadow-xs space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-200 pb-3">
              <div className="flex items-center gap-2">
                <Eye className="w-5 h-5 text-zinc-900" />
                <h2 className="text-base font-black text-zinc-900">Предпросмотр документа КП</h2>
              </div>
              {latestOffer && (
                <span className="text-[11px] font-black px-2.5 py-0.5 rounded-full border border-zinc-900 bg-[#FAF8F2]">
                  Версия {latestOffer.version}
                </span>
              )}
            </div>

            {loadingPdf ? (
              <div className="h-80 flex flex-col items-center justify-center text-zinc-400 gap-2">
                <RefreshCw className="w-6 h-6 animate-spin text-zinc-900" />
                <p className="text-xs font-bold">Формирование PDF-документа...</p>
              </div>
            ) : pdfBlobUrl ? (
              <div className="w-full h-[450px] rounded-2xl overflow-hidden border-2 border-zinc-900 bg-zinc-100">
                <iframe
                  src={`${pdfBlobUrl}#toolbar=0&navpanes=0&scrollbar=0`}
                  title="PDF Preview"
                  className="w-full h-full border-none"
                />
              </div>
            ) : (
              <div className="h-72 flex flex-col items-center justify-center p-6 text-center text-zinc-400 bg-[#FAF8F2] rounded-2xl border-2 border-dashed border-zinc-300 space-y-2">
                <FileText className="w-10 h-10 text-zinc-400" />
                <p className="text-xs font-bold text-zinc-700">PDF-документ будет доступен после формирования</p>
                <p className="text-[11px] text-zinc-500 max-w-xs">
                  Нажмите «{latestOffer ? 'Обновить предложение' : 'Сформировать сделку'}», чтобы создать коммерческое предложение и сгенерировать официальный PDF.
                </p>
              </div>
            )}
          </div>

        </div>

      </div>

    </div>
  );
};
