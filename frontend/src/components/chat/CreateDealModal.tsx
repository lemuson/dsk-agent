import React, { useState } from 'react';
import { X, FileText, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import { ChatSession, Apartment, Deal } from '../../types';
import { dealsApi } from '../../api/deals';
import { chatsApi } from '../../api/chats';
import { formatPrice } from '../../lib/utils';

interface CreateDealModalProps {
  session: ChatSession;
  apartment?: Apartment | null;
  onClose: () => void;
  onSuccess: (deal: Deal) => void;
}

export const CreateDealModal: React.FC<CreateDealModalProps> = ({
  session,
  apartment,
  onClose,
  onSuccess,
}) => {
  const [discountPercent, setDiscountPercent] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const basePrice = apartment?.price || 5000000;
  const discountAmount = (basePrice * discountPercent) / 100;
  const totalPrice = Math.max(0, basePrice - discountAmount);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session.id_user) {
      setErrorMsg('У данного обращения нет зарегистрированного пользователя');
      return;
    }
    const aptId = apartment?.id || session.id_apartment;
    if (!aptId) {
      setErrorMsg('Не выбрана квартира для сделки');
      return;
    }

    setLoading(true);
    setErrorMsg(null);
    try {
      const newDeal = await dealsApi.createDeal({
        id_user: session.id_user,
        id_apartment: aptId,
        id_chat_session: session.id,
        percent_discount: Number(discountPercent),
      });
      onSuccess(newDeal);
    } catch (err: any) {
      setErrorMsg(err.message || 'Ошибка создания сделки');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-xs">
      <div
        className="bg-white rounded-2xl border border-slate-200 w-full max-w-md overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >

        <div className="p-5 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-lg bg-slate-100 border border-slate-200 flex items-center justify-center text-slate-700">
              <FileText className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-bold text-slate-900 text-sm">Оформление сделки</h3>
              <p className="text-xs text-slate-400">Обращение #{session.id}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg border border-slate-200 hover:bg-slate-100 text-slate-500 hover:text-slate-900 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleCreate} className="p-5 space-y-4">

          {errorMsg && (
            <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-xs text-rose-700 flex items-start gap-2">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          <div className="p-3.5 bg-slate-50 rounded-xl border border-slate-100 space-y-1.5 text-xs">
            <div className="flex justify-between">
              <span className="text-slate-500">Клиент ID:</span>
              <span className="font-semibold text-slate-900">{session.id_user || 'Гость'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">Квартира:</span>
              <span className="font-semibold text-slate-900">
                {apartment ? `№${apartment.number} (${apartment.area} м²)` : session.id_apartment || 'Не выбрана'}
              </span>
            </div>
            <div className="flex justify-between border-t border-slate-200/60 pt-1.5">
              <span className="text-slate-500">Базовая стоимость:</span>
              <span className="font-bold text-slate-900">{formatPrice(basePrice)}</span>
            </div>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <label className="font-semibold text-slate-700">
                Скидка менеджера:
              </label>
              <span className="font-bold text-slate-900 bg-slate-100 px-2 py-0.5 rounded">
                {discountPercent}%
              </span>
            </div>

            <input
              type="range"
              min="0"
              max="15"
              step="0.5"
              value={discountPercent}
              onChange={(e) => setDiscountPercent(parseFloat(e.target.value))}
              className="w-full h-1.5 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-slate-900"
            />
            {discountPercent > 5 && (
              <p className="text-[11px] text-amber-700 bg-amber-50 p-2 rounded-lg border border-amber-200">
                Скидки выше 5% потребуют утверждения руководителем.
              </p>
            )}
          </div>

          <div className="p-3 bg-slate-50 rounded-xl border border-slate-100 space-y-1 text-xs">
            <div className="flex justify-between text-slate-500">
              <span>Сумма скидки:</span>
              <span>- {formatPrice(discountAmount)}</span>
            </div>
            <div className="flex justify-between text-sm font-bold text-slate-900 border-t border-slate-200/60 pt-1">
              <span>Итоговая сумма:</span>
              <span>{formatPrice(totalPrice)}</span>
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-white border border-slate-200 hover:bg-slate-100 text-slate-700 text-xs font-semibold rounded-lg transition-colors"
            >
              Отмена
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-5 py-2 bg-slate-900 hover:bg-slate-800 text-white text-xs font-semibold rounded-lg transition-colors flex items-center gap-1.5 disabled:opacity-50"
            >
              {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
              <span>Создать сделку</span>
            </button>
          </div>

        </form>
      </div>
    </div>
  );
};
