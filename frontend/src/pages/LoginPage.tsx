import React, { useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { Building2, LogIn, Lock, Mail, AlertCircle, Loader2 } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

export const LoginPage: React.FC = () => {
  const { login, isAuthenticated, user } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isStaff, setIsStaff] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  React.useEffect(() => {
    if (isAuthenticated && user) {
      if (user.role === 'supervisor') navigate('/supervisor');
      else if (user.role === 'manager') navigate('/manager');
      else navigate('/profile');
    }
  }, [isAuthenticated, user, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) {
      setErrorMsg('Заполните все поля');
      return;
    }

    setLoading(true);
    setErrorMsg(null);
    try {
      const loggedUser = await login(email, password, isStaff);
      const redirect = searchParams.get('redirect');
      if (redirect) {
        navigate(redirect);
      } else if (loggedUser.role === 'supervisor') {
        navigate('/supervisor');
      } else if (loggedUser.role === 'manager') {
        navigate('/manager');
      } else {
        navigate('/profile');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Неверный email или пароль');
    } finally {
      setLoading(false);
    }
  };

  const handleDemoLogin = async (demoEmail: string, isStaffRole: boolean) => {
    setEmail(demoEmail);
    setPassword('Demo123!');
    setIsStaff(isStaffRole);
    setLoading(true);
    setErrorMsg(null);
    try {
      const loggedUser = await login(demoEmail, 'Demo123!', isStaffRole);
      if (loggedUser.role === 'supervisor') navigate('/supervisor');
      else if (loggedUser.role === 'manager') navigate('/manager');
      else navigate('/profile');
    } catch (err: any) {
      setErrorMsg(err.message || 'Ошибка демо-входа');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[85vh] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-6">

        <div className="text-center space-y-2">
          <div className="w-14 h-14 rounded-2xl bg-white border-2 border-zinc-900 text-zinc-900 mx-auto flex items-center justify-center font-bold">
            <Building2 className="w-7 h-7" />
          </div>
          <h1 className="text-2xl font-black tracking-tight text-zinc-900">
            Вход в систему ДСК
          </h1>
          <p className="text-xs font-medium text-zinc-600">
            Личный кабинет покупателя и рабочее место сотрудника
          </p>
        </div>

        <div className="flex bg-[#FAF8F2] p-1.5 rounded-2xl border-2 border-zinc-900 gap-1.5">
          <button
            type="button"
            onClick={() => setIsStaff(false)}
            className={`flex-1 py-2 text-xs font-bold rounded-xl transition-all ${
              !isStaff
                ? 'bg-zinc-900 text-white border-2 border-zinc-900'
                : 'text-zinc-600 hover:text-zinc-900 bg-transparent border-2 border-transparent'
            }`}
          >
            Покупатель
          </button>
          <button
            type="button"
            onClick={() => setIsStaff(true)}
            className={`flex-1 py-2 text-xs font-bold rounded-xl transition-all ${
              isStaff
                ? 'bg-zinc-900 text-white border-2 border-zinc-900'
                : 'text-zinc-600 hover:text-zinc-900 bg-transparent border-2 border-transparent'
            }`}
          >
            Сотрудник ДСК
          </button>
        </div>

        <div className="bg-white p-6 sm:p-7 rounded-2xl border-2 border-zinc-900 space-y-5">

          {errorMsg && (
            <div className="p-3 bg-rose-50 border-2 border-rose-300 text-xs font-medium text-rose-800 rounded-xl flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0 text-rose-600" />
              <span>{errorMsg}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-zinc-700 mb-1.5">
                Электронная почта
              </label>
              <div className="relative">
                <Mail className="w-4 h-4 text-zinc-400 absolute left-3.5 top-3" />
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="name@example.com"
                  className="w-full pl-10 pr-3.5 py-2.5 bg-[#FAF8F2] border-2 border-zinc-300 focus:border-zinc-900 rounded-xl text-xs font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-white transition-all"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-zinc-700 mb-1.5">
                Пароль
              </label>
              <div className="relative">
                <Lock className="w-4 h-4 text-zinc-400 absolute left-3.5 top-3" />
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full pl-10 pr-3.5 py-2.5 bg-[#FAF8F2] border-2 border-zinc-300 focus:border-zinc-900 rounded-xl text-xs font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-white transition-all"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-zinc-900 hover:bg-zinc-800 text-white rounded-xl text-xs font-bold flex items-center justify-center gap-2 transition-all border-2 border-zinc-900 disabled:opacity-50"
            >
              {loading ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <LogIn className="w-4 h-4" />
              )}
              <span>Войти в систему</span>
            </button>
          </form>

          <div className="pt-4 border-t-2 border-zinc-200 space-y-2.5">
            <span className="text-[10px] font-bold text-zinc-500 block text-center uppercase tracking-wider">
              Быстрый демо-вход:
            </span>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => handleDemoLogin('manager@dsk.demo', true)}
                className="p-2.5 bg-[#FAF8F2] border-2 border-zinc-300 hover:border-zinc-900 text-xs font-bold text-zinc-800 rounded-xl transition-all text-center"
              >
                Менеджер
              </button>
              <button
                type="button"
                onClick={() => handleDemoLogin('supervisor@dsk.demo', true)}
                className="p-2.5 bg-[#FAF8F2] border-2 border-zinc-300 hover:border-zinc-900 text-xs font-bold text-zinc-800 rounded-xl transition-all text-center"
              >
                Руководитель
              </button>
            </div>
          </div>

          <div className="text-center pt-2">
            <p className="text-xs font-medium text-zinc-600">
              Нет аккаунта?{' '}
              <Link to="/register" className="text-zinc-900 font-bold hover:underline">
                Зарегистрироваться
              </Link>
            </p>
          </div>

        </div>

      </div>
    </div>
  );
};
