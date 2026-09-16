import React, { useState } from 'react';
import { X, SlidersHorizontal, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import { Building, Role, DiscountPolicy } from '../../types';
import { discountsApi } from '../../api/discounts';

interface DiscountPolicyModalProps {
  buildings: Building[];
  initialBuildingId?: number;
  onClose: () => void;
  onSuccess: (policy: DiscountPolicy) => void;
}

export const DiscountPolicyModal: React.FC<DiscountPolicyModalProps> = ({
  buildings,
  initialBuildingId,
  onClose,
  onSuccess,
}) => {
  const [buildingId, setBuildingId] = useState<number>(
    initialBuildingId || (buildings.length > 0 ? buildings[0].id : 0)
  );
  const [role, setRole] = useState<Role>('manager');
  const [maxDiscountPercent, setMaxDiscountPercent] = useState<string>('5.0');
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!buildingId) {
      setErrorMsg('Выберите корпус');
      return;
    }

    setLoading(true);
    setErrorMsg(null);
    try {
      const newPolicy = await discountsApi.createPolicy({
        building_id: buildingId,
        role,
        max_discount_percent: maxDiscountPercent,
        valid_from: new Date().toISOString(),
      });
      onSuccess(newPolicy);
    } catch (err: any) {
      setErrorMsg(err.message || 'Ошибка обновления политики скидок');
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
              <SlidersHorizontal className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-bold text-slate-900 text-sm">Матрица скидок</h3>
              <p className="text-xs text-slate-400">Лимиты скидок для корпуса</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg border border-slate-200 hover:bg-slate-100 text-slate-500 hover:text-slate-900 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">

          {errorMsg && (
            <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-xs text-rose-700 flex items-start gap-2">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Корпус / Объект
            </label>
            <select
              value={buildingId}
              onChange={(e) => setBuildingId(Number(e.target.value))}
              className="w-full h-9 px-3 bg-slate-50 border border-slate-200 rounded-lg text-xs font-medium text-slate-900 focus:outline-none focus:border-slate-400 focus:bg-white"
            >
              {buildings.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.address} (ID: {b.id}, р-н {b.district})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Роль сотрудника
            </label>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => {
                  setRole('manager');
                  if (parseFloat(maxDiscountPercent) > 7) setMaxDiscountPercent('5.0');
                }}
                className={`py-2 rounded-lg text-xs font-semibold border transition-colors ${
                  role === 'manager'
                    ? 'bg-slate-900 border-slate-900 text-white'
                    : 'bg-slate-50 border-slate-200 text-slate-700 hover:bg-slate-100'
                }`}
              >
                Менеджер
              </button>
              <button
                type="button"
                onClick={() => {
                  setRole('supervisor');
                  if (parseFloat(maxDiscountPercent) <= 5) setMaxDiscountPercent('12.0');
                }}
                className={`py-2 rounded-lg text-xs font-semibold border transition-colors ${
                  role === 'supervisor'
                    ? 'bg-slate-900 border-slate-900 text-white'
                    : 'bg-slate-50 border-slate-200 text-slate-700 hover:bg-slate-100'
                }`}
              >
                Руководитель
              </button>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Максимальный процент скидки (%)
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                min="0"
                max="25"
                step="0.5"
                value={maxDiscountPercent}
                onChange={(e) => setMaxDiscountPercent(e.target.value)}
                className="w-full h-9 px-3 bg-slate-50 border border-slate-200 rounded-lg text-xs font-semibold text-slate-900 focus:outline-none focus:border-slate-400 focus:bg-white"
                required
              />
              <span className="text-xs font-bold text-slate-500 bg-slate-100 px-3 h-9 rounded-lg flex items-center justify-center">
                %
              </span>
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
              <span>Сохранить</span>
            </button>
          </div>

        </form>
      </div>
    </div>
  );
};
