import React, { useState, useEffect } from 'react';
import { MessageSquare, X, ExternalLink, Loader2, User, Phone, Mail, Send, Sparkles } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { chatsApi } from '../../api/chats';
import { ChatSession } from '../../types';
import { ChatRoom } from './ChatRoom';

interface GeneralChatModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const GeneralChatModal: React.FC<GeneralChatModalProps> = ({ isOpen, onClose }) => {
  const { user, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  const [activeSessionId, setActiveSessionId] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const [guestName, setGuestName] = useState('');
  const [guestContact, setGuestContact] = useState('');
  const [guestMessage, setGuestMessage] = useState('');
  const [submittingGuest, setSubmittingGuest] = useState(false);

  useEffect(() => {
    if (!isOpen) return;

    if (isAuthenticated) {
      setLoading(true);
      setErrorMsg(null);

      chatsApi
        .getMySessions()
        .then(async (sessions) => {

          const generalSession = sessions.find((s) => !s.id_apartment);
          if (generalSession) {
            setActiveSessionId(generalSession.id);
          } else {

            const created = await chatsApi.createSession({});
            setActiveSessionId(created.id);
          }
        })
        .catch((err) => {
          console.error('Failed to load/create general chat session:', err);
          setErrorMsg('Не удалось подключиться к чату. Попробуйте обновить страницу.');
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [isOpen, isAuthenticated]);

  if (!isOpen) return null;

  const handleStartGuestChat = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!guestContact.trim() || submittingGuest) return;

    setSubmittingGuest(true);
    setErrorMsg(null);

    const isEmail = guestContact.includes('@');

    try {
      const created = await chatsApi.createSession({
        guest_name: guestName.trim() || 'Гость',
        guest_email: isEmail ? guestContact.trim() : undefined,
        guest_phone: !isEmail ? guestContact.trim() : undefined,
      });

      if (guestMessage.trim()) {
        await chatsApi.sendMessage(created.id, guestMessage.trim(), false);
      }

      setActiveSessionId(created.id);
    } catch (err: any) {
      console.error('Failed to create guest chat:', err);
      setErrorMsg(err.message || 'Ошибка создания обращения. Проверьте контактные данные.');
    } finally {
      setSubmittingGuest(false);
    }
  };

  const handleOpenFullscreen = () => {
    if (activeSessionId) {
      navigate(`/profile?session=${activeSessionId}`);
      onClose();
    } else {
      navigate('/profile');
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-900/60 backdrop-blur-xs p-3 sm:p-4">
      <div className="bg-white rounded-3xl border-2 border-zinc-900 w-full max-w-2xl h-[85vh] sm:h-[80vh] flex flex-col overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-150">

        <div className="h-16 px-5 border-b-2 border-zinc-900 flex items-center justify-between bg-[#FAF8F2] shrink-0">
          <div className="flex items-center gap-3">
            <div className="relative">
              <div className="w-10 h-10 rounded-2xl bg-zinc-900 text-white flex items-center justify-center font-extrabold text-sm border-2 border-zinc-900 shadow-xs">
                <MessageSquare className="w-5 h-5 text-white" />
              </div>
              <span className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 bg-emerald-500 rounded-full border-2 border-white" />
            </div>

            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-sm font-extrabold text-zinc-900">
                  Консультация ДСК
                </h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-extrabold bg-emerald-50 text-emerald-800 border border-emerald-300">
                  Онлайн
                </span>
              </div>
              <p className="text-[11px] text-zinc-500 font-medium">
                Общие вопросы, условия покупки, ипотека и подбор
              </p>
            </div>
          </div>

          <div className="flex items-center gap-1.5">
            {isAuthenticated && activeSessionId && (
              <button
                onClick={handleOpenFullscreen}
                className="p-2 rounded-xl text-zinc-600 hover:text-zinc-900 hover:bg-white border border-transparent hover:border-zinc-900 transition-colors"
                title="Открыть на весь экран в личном кабинете"
              >
                <ExternalLink className="w-4 h-4" />
              </button>
            )}
            <button
              onClick={onClose}
              className="p-2 rounded-xl text-zinc-600 hover:text-zinc-900 hover:bg-white border border-transparent hover:border-zinc-900 transition-colors"
              title="Закрыть окно"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="flex-1 min-h-0 bg-white relative">
          {loading ? (
            <div className="h-full flex flex-col items-center justify-center p-6 space-y-3 text-zinc-500">
              <Loader2 className="w-7 h-7 animate-spin text-zinc-900" />
              <p className="text-xs font-bold uppercase tracking-wider">Подключение к диалогу...</p>
            </div>
          ) : errorMsg ? (
            <div className="h-full flex flex-col items-center justify-center p-6 space-y-3 text-center">
              <div className="p-4 bg-rose-50 border-2 border-zinc-900 rounded-2xl max-w-md text-xs font-bold text-rose-900">
                {errorMsg}
              </div>
              <button
                onClick={() => {
                  setErrorMsg(null);
                  if (isAuthenticated) {
                    setLoading(true);
                    chatsApi.getMySessions().then(async (sessions) => {
                      const gen = sessions.find((s) => !s.id_apartment);
                      if (gen) {
                        setActiveSessionId(gen.id);
                      } else {
                        const s = await chatsApi.createSession({});
                        setActiveSessionId(s.id);
                      }
                      setLoading(false);
                    });
                  }
                }}
                className="px-4 py-2 bg-zinc-900 text-white rounded-xl text-xs font-bold border-2 border-zinc-900"
              >
                Повторить попытку
              </button>
            </div>
          ) : isAuthenticated && activeSessionId ? (
            <div className="h-full p-2">
              <ChatRoom sessionId={activeSessionId} />
            </div>
          ) : !isAuthenticated && !activeSessionId ? (
            <div className="h-full overflow-y-auto p-6 flex flex-col justify-center max-w-md mx-auto">
              <div className="text-center space-y-2 mb-6">
                <div className="w-12 h-12 rounded-2xl bg-[#FAF8F2] border-2 border-zinc-900 text-zinc-900 flex items-center justify-center mx-auto shadow-xs">
                  <MessageSquare className="w-6 h-6" />
                </div>
                <h3 className="text-base font-extrabold text-zinc-900">
                  Задайте вопрос специалисту отдела продаж
                </h3>
                <p className="text-xs text-zinc-600 font-medium">
                  Специалист ДСК проконсультирует вас по всем ЖК, ипотечным программам и акциям.
                </p>
              </div>

              <form onSubmit={handleStartGuestChat} className="space-y-3">
                <div>
                  <label className="block text-[11px] font-extrabold uppercase text-zinc-700 mb-1">
                    Ваше имя
                  </label>
                  <input
                    type="text"
                    value={guestName}
                    onChange={(e) => setGuestName(e.target.value)}
                    placeholder="Иван"
                    className="w-full px-3.5 py-2.5 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-[#FAF8F2]"
                  />
                </div>

                <div>
                  <label className="block text-[11px] font-extrabold uppercase text-zinc-700 mb-1">
                    Телефон или Email <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    value={guestContact}
                    onChange={(e) => setGuestContact(e.target.value)}
                    placeholder="+7 (999) 000-00-00 или mail@example.com"
                    className="w-full px-3.5 py-2.5 bg-white border-2 border-zinc-900 rounded-xl text-xs font-bold text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-[#FAF8F2]"
                  />
                </div>

                <div>
                  <label className="block text-[11px] font-extrabold uppercase text-zinc-700 mb-1">
                    Ваш вопрос
                  </label>
                  <textarea
                    rows={2}
                    value={guestMessage}
                    onChange={(e) => setGuestMessage(e.target.value)}
                    placeholder="Здравствуйте! Хочу узнать подробнее об условиях покупки и спецпредложениях..."
                    className="w-full px-3.5 py-2 bg-white border-2 border-zinc-900 rounded-xl text-xs font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none focus:bg-[#FAF8F2] resize-none"
                  />
                </div>

                <button
                  type="submit"
                  disabled={!guestContact.trim() || submittingGuest}
                  className="w-full py-3 bg-zinc-900 hover:bg-zinc-800 text-white rounded-xl text-xs font-extrabold flex items-center justify-center gap-2 border-2 border-zinc-900 transition-colors disabled:opacity-50 cursor-pointer shadow-xs"
                >
                  {submittingGuest ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                  <span>Начать консультацию</span>
                </button>

                <div className="pt-2 text-center">
                  <p className="text-[11px] text-zinc-500 font-medium">
                    Уже зарегистрированы?{' '}
                    <button
                      type="button"
                      onClick={() => {
                        navigate('/login');
                        onClose();
                      }}
                      className="text-zinc-900 font-bold underline hover:text-zinc-700 cursor-pointer"
                    >
                      Войти в личный кабинет
                    </button>
                  </p>
                </div>
              </form>
            </div>
          ) : activeSessionId ? (
            <div className="h-full p-2">
              <ChatRoom sessionId={activeSessionId} />
            </div>
          ) : null}
        </div>

      </div>
    </div>
  );
};
