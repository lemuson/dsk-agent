import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Search, SlidersHorizontal, RefreshCw, X, Building2, MessageSquare } from 'lucide-react';
import { Apartment, ResidentialComplex, Building, FinishingType } from '../types';
import { apartmentsApi } from '../api/apartments';
import { chatsApi } from '../api/chats';
import { ApartmentCard } from '../components/apartments/ApartmentCard';
import { ApartmentModal } from '../components/apartments/ApartmentModal';
import { useAuth } from '../context/AuthContext';
import { formatPrice } from '../lib/utils';

export const ApartmentsPage: React.FC = () => {
  const { isAuthenticated, user } = useAuth();
  const navigate = useNavigate();
  const { id: urlApartmentId } = useParams<{ id?: string }>();
  const isStaff = user?.role === 'manager' || user?.role === 'supervisor';

  const [apartments, setApartments] = useState<Apartment[]>([]);
  const [complexes, setComplexes] = useState<ResidentialComplex[]>([]);
  const [buildings, setBuildings] = useState<Building[]>([]);
  const [loading, setLoading] = useState(true);

  const [selectedComplexId, setSelectedComplexId] = useState<number | 'all'>('all');
  const [selectedRooms, setSelectedRooms] = useState<number | 'all'>('all');
  const [minPrice, setMinPrice] = useState<number>(0);
  const [maxPrice, setMaxPrice] = useState<number>(15000000);
  const [minArea, setMinArea] = useState<number>(0);
  const [maxArea, setMaxArea] = useState<number>(150);

  const [userSessionsMap, setUserSessionsMap] = useState<Record<number, number>>({});

  const [activeApartment, setActiveApartment] = useState<Apartment | null>(null);

  const loadCatalog = async () => {
    setLoading(true);
    try {
      const data = await apartmentsApi.getAllAvailableApartments();
      setApartments(data.apartments);
      setComplexes(data.complexes);
      setBuildings(data.buildings);

      if (urlApartmentId) {
        const target = data.apartments.find((a) => a.id === Number(urlApartmentId));
        if (target) setActiveApartment(target);
      }

      if (isAuthenticated && !isStaff) {
        try {
          const sessions = await chatsApi.getMySessions();
          const map: Record<number, number> = {};
          for (const s of sessions) {
            if (s.id_apartment && !s.deleted_by_user) {
              map[s.id_apartment] = s.id;
            }
          }
          setUserSessionsMap(map);
        } catch {

        }
      }
    } catch (err) {
      console.error('Failed to load apartments catalog:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCatalog();
  }, [isAuthenticated, isStaff]);

  useEffect(() => {
    if (urlApartmentId && apartments.length > 0) {
      const target = apartments.find((a) => a.id === Number(urlApartmentId));
      if (target) {
        setActiveApartment(target);
      }
    } else if (!urlApartmentId) {
      setActiveApartment(null);
    }
  }, [urlApartmentId, apartments]);

  const handleSelectApartment = (apt: Apartment) => {
    setActiveApartment(apt);
    navigate(`/apartments/${apt.id}`);
  };

  const handleCloseModal = () => {
    setActiveApartment(null);
    navigate('/apartments');
  };

  const handleGoToChat = (sessionId: number) => {
    navigate(`/profile?session=${sessionId}`);
  };

  const handleContactManager = async (apt: Apartment) => {
    if (!isAuthenticated) {
      navigate(`/login?redirect=/apartments/${apt.id}&apt=${apt.id}`);
      return;
    }

    try {
      const session = await chatsApi.createSession({
        id_apartment: apt.id,
      });
      navigate(`/profile?session=${session.id}`);
    } catch (err) {
      console.error('Failed to create chat session for apartment:', err);
      navigate('/profile');
    }
  };

  const handleContactGeneralManager = async () => {
    if (!isAuthenticated) {
      navigate('/login?redirect=/profile');
      return;
    }

    try {
      const session = await chatsApi.createSession({});
      navigate(`/profile?session=${session.id}`);
    } catch (err) {
      console.error('Failed to create general chat session:', err);
      navigate('/profile');
    }
  };

  const filteredApartments = apartments.filter((apt) => {
    if (apt.status !== 'free') {
      return false;
    }
    if (selectedComplexId !== 'all' && apt.building?.residential_complex_id !== selectedComplexId) {
      return false;
    }
    if (selectedRooms !== 'all' && apt.rooms !== selectedRooms) {
      return false;
    }
    if (apt.price < minPrice) {
      return false;
    }
    if (maxPrice > 0 && apt.price > maxPrice) {
      return false;
    }
    if (minArea > 0 && apt.area < minArea) {
      return false;
    }
    if (maxArea > 0 && apt.area > maxArea) {
      return false;
    }
    return true;
  });

  return (
    <div className="min-h-screen py-8 px-4 sm:px-6 lg:px-8 max-w-6xl mx-auto space-y-6">

      <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold text-zinc-900 tracking-tight">
            Каталог квартир
          </h1>
          <p className="text-xs text-zinc-500 font-medium mt-1">
            Все доступные к покупке квартиры в жилых комплексах ДСК
          </p>
        </div>

        {!isStaff && (
          <button
            onClick={handleContactGeneralManager}
            className="px-4 py-2.5 bg-white hover:bg-[#FAF8F2] text-zinc-900 text-xs font-extrabold rounded-2xl border-2 border-zinc-900 transition-colors flex items-center gap-2 shadow-xs cursor-pointer self-start sm:self-auto"
            title="Задать вопрос менеджеру отдела продаж по подбору или ипотеке"
          >
            <MessageSquare className="w-4 h-4 text-zinc-900" />
            <span>Консультация с менеджером</span>
          </button>
        )}
      </div>

      <div className="bg-[#FAF8F2] rounded-2xl border-2 border-zinc-900 p-5 sm:p-6 space-y-4 shadow-xs">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-300 pb-3">
          <div className="flex items-center gap-2">
            <SlidersHorizontal className="w-4 h-4 text-zinc-900" />
            <h2 className="text-sm font-bold text-zinc-900 uppercase tracking-wide">Параметры поиска</h2>
            <span className="text-xs text-zinc-500 font-medium">
              (доступно: <strong className="text-zinc-900">{filteredApartments.length}</strong>)
            </span>
          </div>

          <button
            onClick={() => {
              setSelectedComplexId('all');
              setSelectedRooms('all');
              setMinPrice(0);
              setMaxPrice(15000000);
              setMinArea(0);
              setMaxArea(150);
            }}
            className="text-xs text-zinc-600 hover:text-zinc-900 font-bold flex items-center gap-1 cursor-pointer transition-colors"
          >
            <X className="w-3.5 h-3.5" />
            Сбросить фильтры
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">

          <div>
            <label className="block text-xs font-bold text-zinc-800 mb-1.5 uppercase tracking-wider">
              Жилой комплекс
            </label>
            <select
              value={selectedComplexId}
              onChange={(e) => setSelectedComplexId(e.target.value === 'all' ? 'all' : Number(e.target.value))}
              className="w-full h-10 px-3 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 focus:outline-none"
            >
              <option value="all">Все жилые комплексы</option>
              {complexes.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-bold text-zinc-800 mb-1.5 uppercase tracking-wider">
              Количество комнат
            </label>
            <div className="flex items-center gap-1.5">
              {[
                { label: 'Все', value: 'all' },
                { label: '1к', value: 1 },
                { label: '2к', value: 2 },
                { label: '3к', value: 3 },
              ].map((r) => (
                <button
                  key={r.value.toString()}
                  onClick={() => setSelectedRooms(r.value as any)}
                  className={`flex-1 h-10 text-xs font-bold rounded-xl border-2 transition-all cursor-pointer ${
                    selectedRooms === r.value
                      ? 'bg-zinc-900 border-zinc-900 text-white'
                      : 'bg-white border-zinc-900 text-zinc-900 hover:bg-zinc-100'
                  }`}
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs font-bold text-zinc-800 mb-1.5 uppercase tracking-wider">
              Цена, ₽
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                placeholder="от 0"
                step="100000"
                value={minPrice || ''}
                onChange={(e) => setMinPrice(Number(e.target.value) || 0)}
                className="w-1/2 h-10 px-3 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 focus:outline-none placeholder-zinc-400"
              />
              <span className="text-zinc-900 font-bold text-xs">—</span>
              <input
                type="number"
                placeholder="до 15 млн"
                step="500000"
                value={maxPrice || ''}
                onChange={(e) => setMaxPrice(Number(e.target.value) || 0)}
                className="w-1/2 h-10 px-3 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 focus:outline-none placeholder-zinc-400"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-bold text-zinc-800 mb-1.5 uppercase tracking-wider">
              Площадь, м²
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                placeholder="от 20"
                step="5"
                value={minArea || ''}
                onChange={(e) => setMinArea(Number(e.target.value) || 0)}
                className="w-1/2 h-10 px-3 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 focus:outline-none placeholder-zinc-400"
              />
              <span className="text-zinc-900 font-bold text-xs">—</span>
              <input
                type="number"
                placeholder="до 150"
                step="5"
                value={maxArea || ''}
                onChange={(e) => setMaxArea(Number(e.target.value) || 0)}
                className="w-1/2 h-10 px-3 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 focus:outline-none placeholder-zinc-400"
              />
            </div>
          </div>

        </div>

      </div>

      {loading ? (
        <div className="py-16 text-center text-zinc-500 flex flex-col items-center gap-2">
          <RefreshCw className="w-6 h-6 animate-spin text-zinc-900" />
          <p className="text-xs font-bold uppercase tracking-wider">Загрузка каталога квартир...</p>
        </div>
      ) : filteredApartments.length === 0 ? (
        <div className="p-12 text-center bg-white rounded-2xl border-2 border-zinc-900 text-zinc-600 space-y-2">
          <Building2 className="w-10 h-10 mx-auto text-zinc-400" />
          <h3 className="text-base font-bold text-zinc-900">Доступных квартир по заданным параметрам не найдено</h3>
          <p className="text-xs text-zinc-500">Попробуйте расширить диапазон цен или изменить параметры комнатности.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {filteredApartments.map((apt) => (
            <ApartmentCard
              key={apt.id}
              apartment={apt}
              existingSessionId={userSessionsMap[apt.id]}
              onSelect={handleSelectApartment}
              onContact={isStaff ? undefined : handleContactManager}
              onGoToChat={handleGoToChat}
            />
          ))}
        </div>
      )}

      {activeApartment && (
        <ApartmentModal
          apartment={activeApartment}
          existingSessionId={userSessionsMap[activeApartment.id]}
          onClose={handleCloseModal}
          onContact={isStaff ? undefined : handleContactManager}
          onGoToChat={handleGoToChat}
        />
      )}

    </div>
  );
};
