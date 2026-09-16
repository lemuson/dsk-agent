import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { SlidersHorizontal, Plus, ArrowLeft, RefreshCw } from 'lucide-react';
import { Building, DiscountPolicy, ResidentialComplex } from '../types';
import { apartmentsApi } from '../api/apartments';
import { discountsApi } from '../api/discounts';
import { DiscountPolicyModal } from '../components/discounts/DiscountPolicyModal';
import { formatDate } from '../lib/utils';

export const SupervisorDiscountsPage: React.FC = () => {
  const [complexes, setComplexes] = useState<ResidentialComplex[]>([]);
  const [buildings, setBuildings] = useState<Building[]>([]);
  const [selectedBuildingId, setSelectedBuildingId] = useState<number>(0);
  const [policies, setPolicies] = useState<DiscountPolicy[]>([]);

  const [loading, setLoading] = useState(true);
  const [loadingPolicies, setLoadingPolicies] = useState(false);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    const init = async () => {
      setLoading(true);
      try {
        const catalog = await apartmentsApi.getAllAvailableApartments();
        setComplexes(catalog.complexes);
        setBuildings(catalog.buildings);
        if (catalog.buildings.length > 0) {
          setSelectedBuildingId(catalog.buildings[0].id);
        }
      } catch (e) {
        console.error('Failed to load buildings:', e);
      } finally {
        setLoading(false);
      }
    };
    init();
  }, []);

  const loadPolicies = async (bId: number) => {
    if (!bId) return;
    setLoadingPolicies(true);
    try {
      const data = await discountsApi.getPoliciesByBuilding(bId);
      setPolicies(data || []);
    } catch (e) {
      console.warn('Failed to load policies:', e);
      setPolicies([]);
    } finally {
      setLoadingPolicies(false);
    }
  };

  useEffect(() => {
    if (selectedBuildingId) {
      loadPolicies(selectedBuildingId);
    }
  }, [selectedBuildingId]);

  const selectedBuilding = buildings.find((b) => b.id === selectedBuildingId);

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">

      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <Link
            to="/supervisor"
            className="inline-flex items-center gap-1.5 text-xs font-bold text-zinc-500 hover:text-zinc-900 mb-2 uppercase tracking-wider"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            Назад в дашборд
          </Link>
          <h1 className="text-2xl font-extrabold text-zinc-900 flex items-center gap-2">
            <SlidersHorizontal className="w-6 h-6 text-zinc-900" />
            Управление матрицей скидок
          </h1>
          <p className="text-xs text-zinc-500 font-medium mt-0.5">
            Установка предельных лимитов скидок для менеджеров и руководителей по корпусам
          </p>
        </div>

        <button
          onClick={() => setShowModal(true)}
          className="px-5 py-2.5 bg-zinc-900 hover:bg-zinc-800 text-white rounded-xl text-xs font-bold flex items-center gap-2 border-2 border-zinc-900 transition-all cursor-pointer shadow-xs"
        >
          <Plus className="w-4 h-4" />
          Новая версия политики
        </button>
      </div>

      <div className="bg-[#FAF8F2] p-5 rounded-3xl border-2 border-zinc-900 space-y-3 shadow-xs">
        <label className="block text-xs font-extrabold text-zinc-900 uppercase tracking-wider">
          Выберите объект / корпус:
        </label>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {buildings.map((b) => {
            const isSelected = selectedBuildingId === b.id;
            return (
              <button
                key={b.id}
                onClick={() => setSelectedBuildingId(b.id)}
                className={`p-3.5 rounded-2xl border-2 text-left transition-all cursor-pointer ${
                  isSelected
                    ? 'bg-zinc-900 border-zinc-900 text-white shadow-xs'
                    : 'bg-white border-zinc-900 text-zinc-900 hover:bg-zinc-100'
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-extrabold truncate">{b.address}</span>
                  <span className={`text-[10px] font-extrabold px-2 py-0.5 rounded-full border ${
                    isSelected ? 'bg-zinc-800 text-zinc-200 border-zinc-700' : 'bg-[#FAF8F2] text-zinc-900 border-zinc-900'
                  }`}>
                    ID {b.id}
                  </span>
                </div>
                <p className={`text-[11px] font-medium truncate ${isSelected ? 'text-zinc-300' : 'text-zinc-500'}`}>
                  р-н {b.district} • {b.floors_count} этажей
                </p>
              </button>
            );
          })}
        </div>
      </div>

      <div className="bg-white p-6 rounded-3xl border-2 border-zinc-900 space-y-4 shadow-xs">
        <div className="flex items-center justify-between border-b border-zinc-300 pb-2">
          <div>
            <h3 className="text-base font-extrabold text-zinc-900">
              Политики скидок: {selectedBuilding?.address || 'Корпус'}
            </h3>
            <p className="text-xs text-zinc-500 font-medium">
              Серверная история версий и действующие ограничения
            </p>
          </div>
          <span className="text-xs text-zinc-900 font-extrabold">
            {policies.length} версий
          </span>
        </div>

        {loadingPolicies ? (
          <div className="py-8 text-center text-zinc-400">
            <RefreshCw className="w-5 h-5 animate-spin mx-auto text-zinc-900 mb-2" />
            <p className="text-xs font-bold uppercase tracking-wider">Загрузка матрицы...</p>
          </div>
        ) : policies.length === 0 ? (
          <div className="py-8 text-center text-zinc-500 bg-[#FAF8F2] rounded-2xl border-2 border-zinc-900 space-y-1">
            <p className="text-xs font-extrabold text-zinc-900">Для данного корпуса действуют стандартные настройки</p>
            <p className="text-[11px] text-zinc-500 font-medium">Нажмите «Новая версия политики», чтобы задать индивидуальные лимиты.</p>
          </div>
        ) : (
          <div className="overflow-x-auto rounded-2xl border-2 border-zinc-900">
            <table className="w-full text-left text-xs border-collapse bg-white">
              <thead>
                <tr className="border-b-2 border-zinc-900 bg-[#FAF8F2] text-zinc-900 font-extrabold uppercase tracking-wider text-[10px]">
                  <th className="py-3 px-4">Версия</th>
                  <th className="py-3 px-4">Роль</th>
                  <th className="py-3 px-4">Макс. скидка</th>
                  <th className="py-3 px-4">Действует с</th>
                  <th className="py-3 px-4">Действует по</th>
                  <th className="py-3 px-4">Статус</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200 font-semibold text-zinc-800">
                {policies.map((pol) => {
                  const isActive = !pol.valid_to;
                  return (
                    <tr key={pol.id} className="hover:bg-[#FAF8F2]/60 transition-colors">
                      <td className="py-3 px-4 font-extrabold text-zinc-900">v{pol.version} (ID: {pol.id})</td>
                      <td className="py-3 px-4">
                        <span className="px-2.5 py-0.5 rounded-full text-[10px] font-extrabold bg-[#FAF8F2] border border-zinc-900 text-zinc-900">
                          {pol.role === 'supervisor' ? 'Руководитель' : 'Менеджер'}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-sm font-extrabold text-zinc-900">
                        {pol.max_discount_percent}%
                      </td>
                      <td className="py-3 px-4 text-zinc-700">{formatDate(pol.valid_from)}</td>
                      <td className="py-3 px-4 text-zinc-400">{pol.valid_to ? formatDate(pol.valid_to) : 'Бессрочно'}</td>
                      <td className="py-3 px-4">
                        <span
                          className={`px-2.5 py-0.5 rounded-full text-[10px] font-extrabold border ${
                            isActive
                              ? 'bg-[#EBF7EE] text-emerald-900 border-zinc-900'
                              : 'bg-zinc-100 text-zinc-700 border-zinc-900'
                          }`}
                        >
                          {isActive ? 'Действующая' : 'Архивная'}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {showModal && (
        <DiscountPolicyModal
          buildings={buildings}
          initialBuildingId={selectedBuildingId}
          onClose={() => setShowModal(false)}
          onSuccess={(newPol) => {
            setShowModal(false);
            if (selectedBuildingId) loadPolicies(selectedBuildingId);
          }}
        />
      )}

    </div>
  );
};
