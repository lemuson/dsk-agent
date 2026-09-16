import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Building2, UserPlus, Lock, Mail, User, AlertCircle, Loader2 } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

export const RegisterPage: React.FC = () => {
  const { register } = useAuth();
  const navigate = useNavigate();

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !email || !password) {
      setErrorMsg('Пожалуйста, заполните все обязательные поля');
      return;
    }
    if (password !== confirmPassword) {
      setErrorMsg('Пароли не совпадают');
      return;
    }

    setLoading(true);
    setErrorMsg(null);
    try {
      await register(name, email, password);
      navigate('/apartments');
    } catch (err: any) {
      setErrorMsg(err.message || 'Ошибка регистрации аккаунта');
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
            Регистрация покупателя
          </h1>
          <p className="text-xs font-medium text-zinc-600">
            Создайте аккаунт для бронирования и прямого чата с менеджером
          </p>
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
                ФИО / Ваше имя
              </label>
              <div className="relative">
                <User className="w-4 h-4 text-zinc-400 absolute left-3.5 top-3" />
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Иван Иванов"
                  className="w-full pl-10 pr-3.5 py-2.5 bg-[#FAF8F2] border-2 border-zinc-300 focus:border-zinc-900 rounded-xl text-xs font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-white transition-all"
                />
              </div>
            </div>

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

            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-zinc-700 mb-1.5">
                Подтверждение пароля
              </label>
              <div className="relative">
                <Lock className="w-4 h-4 text-zinc-400 absolute left-3.5 top-3" />
                <input
                  type="password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
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
                <UserPlus className="w-4 h-4" />
              )}
              <span>Создать аккаунт</span>
            </button>
          </form>

          <div className="text-center pt-2">
            <p className="text-xs font-medium text-zinc-600">
              Уже есть аккаунт?{' '}
              <Link to="/login" className="text-zinc-900 font-bold hover:underline">
                Войти в систему
              </Link>
            </p>
          </div>

        </div>

      </div>
    </div>
  );
};
