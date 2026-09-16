import React from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { Building2, MessageSquare, User as UserIcon, LogOut, Sparkles, SlidersHorizontal } from 'lucide-react';
import { useAuth } from '../../context/AuthContext';
import { NotificationBell } from '../notifications/NotificationBell';

export const Navbar: React.FC = () => {
  const { user, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const isActive = (path: string) => location.pathname === path;
  const isStaff = user?.role === 'manager' || user?.role === 'supervisor';

  return (
    <>
      <header className="sticky top-0 z-40 bg-[#F6F4EE]/90 backdrop-blur-md border-b-2 border-zinc-900 py-3">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="bg-white border-2 border-zinc-900 rounded-2xl px-4 sm:px-6 h-14 flex items-center justify-between shadow-xs">

            <div className="flex items-center gap-6">
              <Link to="/" className="flex items-center gap-2.5 group">
                <div className="w-8 h-8 rounded-xl bg-zinc-900 flex items-center justify-center text-white font-bold text-sm">
                  <Building2 className="w-4 h-4" />
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="font-extrabold text-base tracking-tight text-zinc-900">ДСК</span>
                </div>
              </Link>

              <nav className="hidden md:flex items-center gap-1.5">
                <Link
                  to="/apartments"
                  className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all border ${
                    isActive('/apartments') || isActive('/') || location.pathname.startsWith('/apartments')
                      ? 'bg-zinc-900 text-white border-zinc-900'
                      : 'text-zinc-700 hover:bg-[#FAF8F2] border-transparent hover:border-zinc-900'
                  }`}
                >
                  Каталог квартир
                </Link>

                {isAuthenticated && user?.role === 'user' && (
                  <Link
                    to="/profile"
                    className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all border ${
                      isActive('/profile')
                        ? 'bg-zinc-900 text-white border-zinc-900'
                        : 'text-zinc-700 hover:bg-[#FAF8F2] border-transparent hover:border-zinc-900'
                    }`}
                  >
                    Мои чаты и сделки
                  </Link>
                )}

                {isAuthenticated && user?.role === 'manager' && (
                  <Link
                    to="/manager"
                    className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all border ${
                      isActive('/manager')
                        ? 'bg-zinc-900 text-white border-zinc-900'
                        : 'text-zinc-700 hover:bg-[#FAF8F2] border-transparent hover:border-zinc-900'
                    }`}
                  >
                    Рабочее место менеджера
                  </Link>
                )}

                {isAuthenticated && user?.role === 'supervisor' && (
                  <>
                    <Link
                      to="/supervisor"
                      className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all border ${
                        isActive('/supervisor')
                          ? 'bg-zinc-900 text-white border-zinc-900'
                          : 'text-zinc-700 hover:bg-[#FAF8F2] border-transparent hover:border-zinc-900'
                      }`}
                    >
                      Обзор отдела
                    </Link>
                    <Link
                      to="/supervisor/discounts"
                      className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all border flex items-center gap-1.5 ${
                        isActive('/supervisor/discounts')
                          ? 'bg-zinc-900 text-white border-zinc-900'
                          : 'text-zinc-700 hover:bg-[#FAF8F2] border-transparent hover:border-zinc-900'
                      }`}
                    >
                      <SlidersHorizontal className="w-3.5 h-3.5" />
                      Матрица скидок
                    </Link>
                  </>
                )}
              </nav>
            </div>

            <div className="flex items-center gap-2.5">

              {isAuthenticated ? (
                <>
                  <NotificationBell />

                  <div className="flex items-center gap-2">
                    <div className="w-8 h-8 rounded-full bg-[#FAF8F2] border border-zinc-900 flex items-center justify-center font-bold text-xs text-zinc-900">
                      {user?.name ? user.name.slice(0, 2).toUpperCase() : <UserIcon className="w-3.5 h-3.5" />}
                    </div>
                    <div className="hidden lg:block text-left">
                      <p className="text-xs font-bold text-zinc-900 leading-tight truncate max-w-[130px]">
                        {user?.name || user?.email}
                      </p>
                    </div>
                  </div>

                  <button
                    onClick={handleLogout}
                    className="w-8 h-8 rounded-xl border border-zinc-300 hover:border-zinc-900 hover:bg-[#FAF8F2] flex items-center justify-center text-zinc-600 hover:text-zinc-900 transition-colors"
                    title="Выйти из аккаунта"
                  >
                    <LogOut className="w-3.5 h-3.5" />
                  </button>
                </>
              ) : (
                <div className="flex items-center gap-2">
                  <Link
                    to="/login"
                    className="px-3 py-1.5 text-xs font-bold text-zinc-800 hover:bg-[#FAF8F2] border border-transparent hover:border-zinc-900 rounded-xl transition-colors"
                  >
                    Войти
                  </Link>
                  <Link
                    to="/register"
                    className="px-3.5 py-1.5 bg-zinc-900 hover:bg-zinc-800 text-white text-xs font-bold rounded-xl border border-zinc-900 transition-colors"
                  >
                    Регистрация
                  </Link>
                </div>
              )}
            </div>

          </div>
        </div>
      </header>
    </>
  );
};
