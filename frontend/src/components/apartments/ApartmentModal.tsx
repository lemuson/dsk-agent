import React, { useEffect, useState } from 'react';
import { X, Building, MapPin, Layers, CheckCircle2, Clock, AlertTriangle, ShieldCheck } from 'lucide-react';
import { Apartment, ConstructionProgress } from '../../types';
import { apartmentsApi } from '../../api/apartments';
import { formatPrice, formatArea, getRoomsLabel, getFinishingLabel, getApartmentStatusBadge } from '../../lib/utils';

interface ApartmentModalProps {
  apartment: Apartment | null;
  existingSessionId?: number;
  onClose: () => void;
  onContact?: (apt: Apartment) => void;
  onGoToChat?: (sessionId: number) => void;
}

export const ApartmentModal: React.FC<ApartmentModalProps> = ({
  apartment,
  existingSessionId,
  onClose,
  onContact,
  onGoToChat,
}) => {
  const [progress, setProgress] = useState<ConstructionProgress[]>([]);
  const [loadingProgress, setLoadingProgress] = useState(false);

  useEffect(() => {
    if (apartment?.building_id) {
      setLoadingProgress(true);
      apartmentsApi
        .getProgressByBuilding(apartment.building_id)
        .then(setProgress)
        .catch(() => setProgress([]))
        .finally(() => setLoadingProgress(false));
    }
  }, [apartment]);

  if (!apartment) return null;

  const statusBadge = getApartmentStatusBadge(apartment.status);
  const pricePerSqm = apartment.area > 0 ? Math.round(apartment.price / apartment.area) : 0;

  const stageLabels: Record<string, string> = {
    excavation: 'Земляные работы и котлован',
    foundation: 'Фундамент и нулевой цикл',
    frame: 'Возведение монолитного каркаса',
    roofing: 'Кровельные и фасадные работы',
    finishing: 'Инженерные сети и отделка',
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-900/60 backdrop-blur-xs">
      <div
        className="bg-white rounded-3xl border-2 border-zinc-900 w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >

        <div className="p-5 sm:p-6 border-b-2 border-zinc-900 flex items-start justify-between bg-[#FAF8F2]">
          <div>
            <div className="flex items-center gap-2 mb-1.5">
              <span className="text-xs font-bold text-zinc-600 uppercase tracking-wider">
                {apartment.complex?.name || 'Жилой комплекс ДСК'}
              </span>
              <span className={`px-2.5 py-0.5 text-[10px] font-extrabold rounded-full border-2 ${
                apartment.status === 'free'
                  ? 'bg-[#EBF7EE] text-emerald-900 border-zinc-900'
                  : 'bg-zinc-100 text-zinc-800 border-zinc-900'
              }`}>
                {statusBadge.label}
              </span>
            </div>
            <h2 className="text-2xl font-extrabold text-zinc-900">
              {getRoomsLabel(apartment.rooms)}, кв. №{apartment.number}
            </h2>
            <p className="text-xs text-zinc-600 font-medium flex items-center gap-1 mt-1">
              <MapPin className="w-3.5 h-3.5 text-zinc-500" />
              {apartment.building?.address || apartment.complex?.address}, р-н {apartment.building?.district || 'Воронеж'}
            </p>
          </div>

          <button
            onClick={onClose}
            className="w-9 h-9 rounded-xl border-2 border-zinc-900 hover:bg-zinc-100 text-zinc-900 flex items-center justify-center transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-5 sm:p-6 overflow-y-auto space-y-5">

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="p-3.5 bg-[#FAF8F2] rounded-xl border-2 border-zinc-900">
              <p className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Общая площадь</p>
              <p className="text-sm font-extrabold text-zinc-900 mt-1">{formatArea(apartment.area)}</p>
            </div>
            <div className="p-3.5 bg-[#FAF8F2] rounded-xl border-2 border-zinc-900">
              <p className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Этаж</p>
              <p className="text-sm font-extrabold text-zinc-900 mt-1">
                {apartment.floor} из {apartment.building?.floors_count || '—'}
              </p>
            </div>
            <div className="p-3.5 bg-[#FAF8F2] rounded-xl border-2 border-zinc-900">
              <p className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Тип отделки</p>
              <p className="text-xs font-extrabold text-zinc-900 mt-1 truncate">{getFinishingLabel(apartment.type_finishing)}</p>
            </div>
            <div className="p-3.5 bg-[#FAF8F2] rounded-xl border-2 border-zinc-900">
              <p className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Материал стен</p>
              <p className="text-xs font-extrabold text-zinc-900 mt-1 capitalize">{apartment.building?.type_wall_material || 'Панель'}</p>
            </div>
          </div>

          <div className="p-4 sm:p-5 bg-[#FAF8F2] rounded-2xl border-2 border-zinc-900 flex items-center justify-between">
            <div>
              <p className="text-xs text-zinc-500 font-bold uppercase tracking-wider">Стоимость застройщика</p>
              <p className="text-2xl sm:text-3xl font-extrabold text-zinc-900 mt-0.5">{formatPrice(apartment.price)}</p>
              <p className="text-xs text-zinc-500 font-medium">{formatPrice(pricePerSqm)} за м²</p>
            </div>
          </div>

          {apartment.building?.readiness_percent !== undefined && (
            <div className="space-y-2.5">
              <div className="flex items-center justify-between">
                <h4 className="text-xs font-bold text-zinc-900 uppercase tracking-wider flex items-center gap-1.5">
                  <Building className="w-4 h-4 text-zinc-700" />
                  Ход строительства
                </h4>
                <span className="text-xs font-extrabold text-zinc-900">
                  {apartment.building.readiness_percent}% готово
                </span>
              </div>

              <div className="w-full bg-zinc-200 rounded-full h-3 border-2 border-zinc-900 overflow-hidden">
                <div
                  className="bg-zinc-900 h-full transition-all duration-300"
                  style={{ width: `${apartment.building.readiness_percent}%` }}
                />
              </div>

              {progress.length > 0 && (
                <div className="space-y-2 pt-2">
                  {progress.map((stage) => (
                    <div key={stage.id} className="p-3 bg-white rounded-xl border-2 border-zinc-900 flex items-center justify-between text-xs">
                      <div className="flex items-center gap-2.5">
                        {stage.status === 'completed' ? (
                          <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                        ) : stage.status === 'delayed' ? (
                          <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0" />
                        ) : (
                          <Clock className="w-4 h-4 text-zinc-400 shrink-0" />
                        )}
                        <div>
                          <span className="font-bold text-zinc-900">
                            {stageLabels[stage.stage_name] || stage.stage_name}
                          </span>
                          {stage.delay_reason && (
                            <p className="text-[10px] text-amber-700 font-bold">{stage.delay_reason}</p>
                          )}
                        </div>
                      </div>
                      <span className="font-extrabold text-zinc-900">
                        {stage.completion_percentage ?? (stage.status === 'completed' ? 100 : 0)}%
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

        </div>

        <div className="p-4 sm:p-5 border-t-2 border-zinc-900 bg-[#FAF8F2] flex items-center justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2.5 bg-white border-2 border-zinc-900 hover:bg-zinc-100 text-zinc-900 text-xs font-bold rounded-xl transition-colors cursor-pointer"
          >
            Закрыть
          </button>
          {existingSessionId && onGoToChat ? (
            <button
              onClick={() => {
                onClose();
                onGoToChat(existingSessionId);
              }}
              className="px-6 py-2.5 bg-[#FAF8F2] hover:bg-zinc-100 border-2 border-zinc-900 text-zinc-900 text-xs font-bold rounded-xl transition-colors cursor-pointer flex items-center gap-1.5 shadow-xs"
            >
              Перейти в чат
            </button>
          ) : onContact ? (
            <button
              onClick={() => {
                onClose();
                onContact(apartment);
              }}
              className="px-6 py-2.5 bg-zinc-900 hover:bg-zinc-800 text-white text-xs font-bold rounded-xl border-2 border-zinc-900 transition-colors cursor-pointer"
            >
              Оставить заявку
            </button>
          ) : null}
        </div>

      </div>
    </div>
  );
};
