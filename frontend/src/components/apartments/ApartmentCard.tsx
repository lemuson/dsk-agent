import React from 'react';
import { MapPin, Building2, MessageSquare } from 'lucide-react';
import { Apartment } from '../../types';
import { formatPrice, formatArea, getRoomsLabel, getApartmentStatusBadge } from '../../lib/utils';

interface ApartmentCardProps {
  apartment: Apartment;
  existingSessionId?: number;
  onSelect: (apt: Apartment) => void;
  onContact?: (apt: Apartment) => void;
  onGoToChat?: (sessionId: number) => void;
}

export const ApartmentCard: React.FC<ApartmentCardProps> = ({
  apartment,
  existingSessionId,
  onSelect,
  onContact,
  onGoToChat,
}) => {
  const statusBadge = getApartmentStatusBadge(apartment.status);
  const pricePerSqm = apartment.area > 0 ? Math.round(apartment.price / apartment.area) : 0;

  return (
    <div
      onClick={() => onSelect(apartment)}
      className="bg-white border-2 border-zinc-900 rounded-2xl p-4 sm:p-5 hover:border-black transition-all cursor-pointer flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 shadow-xs hover:bg-[#FAF8F2]/40"
    >

      <div className="flex items-start gap-4 flex-1 min-w-0">
        <div className="w-14 h-14 rounded-2xl bg-[#FAF8F2] flex flex-col items-center justify-center flex-shrink-0 text-zinc-900 border-2 border-zinc-900">
          <span className="text-base font-extrabold leading-none">№{apartment.number}</span>
          <span className="text-[10px] text-zinc-600 font-bold uppercase tracking-wider mt-0.5">кв.</span>
        </div>

        <div className="space-y-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-base font-extrabold text-zinc-900 leading-tight">
              {getRoomsLabel(apartment.rooms)}, {formatArea(apartment.area)}
            </h3>
            <span className={`px-2.5 py-0.5 text-[10px] font-extrabold rounded-full border-2 ${
              apartment.status === 'free'
                ? 'bg-[#EBF7EE] text-emerald-900 border-zinc-900'
                : 'bg-zinc-100 text-zinc-800 border-zinc-900'
            }`}>
              {statusBadge.label}
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-zinc-600">
            <span className="font-bold text-zinc-900">
              {apartment.complex?.name || 'Жилой комплекс ДСК'}
            </span>
            <span>•</span>
            <span className="flex items-center gap-1 text-zinc-600 font-medium">
              <MapPin className="w-3.5 h-3.5 text-zinc-500" />
              {apartment.building?.address || apartment.complex?.address || 'Воронеж'}
              {apartment.building?.district && ` (р-н ${apartment.building.district})`}
            </span>
            <span>•</span>
            <span className="text-zinc-600 font-medium">
              {apartment.floor} этаж из {apartment.building?.floors_count || '—'}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between sm:justify-end gap-6 w-full sm:w-auto pt-3 sm:pt-0 border-t sm:border-t-0 border-zinc-200">
        <div className="text-left sm:text-right">
          <p className="text-lg font-extrabold text-zinc-900 leading-tight">
            {formatPrice(apartment.price)}
          </p>
          <p className="text-xs text-zinc-500 font-semibold mt-0.5">
            {formatPrice(pricePerSqm)} / м²
          </p>
        </div>

        {existingSessionId && onGoToChat ? (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onGoToChat(existingSessionId);
            }}
            className="px-5 py-2.5 bg-[#FAF8F2] hover:bg-zinc-100 text-zinc-900 text-xs font-bold rounded-xl border-2 border-zinc-900 transition-all whitespace-nowrap flex-shrink-0 cursor-pointer flex items-center gap-1.5 shadow-xs"
          >
            <MessageSquare className="w-3.5 h-3.5" />
            <span>Перейти в чат</span>
          </button>
        ) : onContact ? (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onContact(apartment);
            }}
            className="px-5 py-2.5 bg-zinc-900 hover:bg-zinc-800 text-white text-xs font-bold rounded-xl border-2 border-zinc-900 transition-all whitespace-nowrap flex-shrink-0 cursor-pointer"
          >
            Оставить заявку
          </button>
        ) : null}
      </div>
    </div>
  );
};
