import React, { useEffect, useRef, useState } from 'react';
import {
  AlertCircle,
  ArrowLeft,
  BarChart3,
  Bot,
  Building2,
  Check,
  ChevronLeft,
  ChevronRight,
  Copy,
  FileText,
  Loader2,
  MessageSquareText,
  Send,
  Sparkles,
  UserRound,
  UserRoundSearch,
  X,
} from 'lucide-react';
import { aiApi } from '../../api/ai';
import { DialogAnalysisResult, ReplyAssistResult, Message } from '../../types';
import { formatPrice } from '../../lib/utils';

interface ManagerAIPanelProps {
  dealId?: number;
  sessionId: number;
  clientName?: string;
  apartmentNumber?: string;
  basePrice?: number;
  chatMessages?: Message[];
  onApplyReplyText?: (text: string) => void;
  onClose?: () => void;
}

type ChatMode = 'negotiation' | 'analysis' | 'reply' | 'risks' | 'offer' | 'general';
type MessageRole = 'user' | 'assistant' | 'system';

interface AIMessage {
  id: string;
  role: MessageRole;
  text: string;
  mode: ChatMode;
  createdAt: string;
  agent?: string;
  intent?: string;
  error?: boolean;
  analysis?: DialogAnalysisResult;
  replyAssist?: ReplyAssistResult;
}

interface QuickQuestion {
  label: string;
  prompt: string;
}

interface CategoryDef {
  id: ChatMode;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  questions: QuickQuestion[];
}

const CATEGORIES: CategoryDef[] = [
  {
    id: 'negotiation',
    label: 'Переговоры',
    icon: MessageSquareText,
    questions: [
      {
        label: 'Возражение по цене',
        prompt: 'Клиент считает цену завышенной или говорит, что у конкурентов дешевле. Подготовь аргументы с учётом характеристик нашей квартиры и текущего диалога.',
      },
      {
        label: 'Сравнить с конкурентами',
        prompt: 'Сравни предложение по нашему объекту с конкурентами в этой локации, опираясь на факты, транспортную доступность и преимущества комплекса.',
      },
      {
        label: 'Ключевые аргументы',
        prompt: 'Подготовь ключевые аргументы для переговоров с клиентом по этой квартире, планировке и локации.',
      },
      {
        label: 'Отработать сомнения',
        prompt: 'Как убедительно развеять сомнения клиента из переписки и мягко подвести его к бронированию или просмотру?',
      },
    ],
  },
  {
    id: 'analysis',
    label: 'Анализ',
    icon: UserRoundSearch,
    questions: [
      {
        label: 'Полный разбор потребностей',
        prompt: 'Проанализируй всю текущую открытую переписку: выдели подтверждённые потребности, бюджет, сомнения, ключевые критерии выбора и предложи следующие шаги менеджера.',
      },
      {
        label: 'Скрытые возражения',
        prompt: 'Какие скрытые сомнения или неозвученные потребности есть у клиента в этом диалоге, и на что менеджеру стоит обратить внимание?',
      },
      {
        label: 'Тактика общения',
        prompt: 'На основе диалога сформулируй оптимальную тактику дальнейшего общения с клиентом.',
      },
    ],
  },
  {
    id: 'reply',
    label: 'Черновик ответа',
    icon: Bot,
    questions: [
      {
        label: 'Ответ на последнее сообщение',
        prompt: 'Подготовь убедительный, вежливый и продающий ответ на последнее сообщение клиента с учётом целевой квартиры и контекста диалога.',
      },
      {
        label: 'Пригласить на просмотр/бронь',
        prompt: 'Сформируй черновик ответа клиенту с мягким предложением забронировать квартиру или записаться на показ в удобное время.',
      },
      {
        label: 'Уточнить параметры поиска',
        prompt: 'Подготовь короткое сообщение клиенту с уточняющими вопросами по бюджету, форме оплаты и срокам покупки.',
      },
    ],
  },
  {
    id: 'risks',
    label: 'Риски застройки',
    icon: BarChart3,
    questions: [
      {
        label: 'Статус возведения дома',
        prompt: 'Проанализируй строительные риски и статус возведения дома для этого объекта.',
      },
      {
        label: 'Сроки сдачи',
        prompt: 'Что ответить клиенту о текущих сроках сдачи и гарантиях застройщика ДСК?',
      },
      {
        label: 'Надёжность ДСК',
        prompt: 'Сформулируй ключевые факты о надёжности ДСК для снятия тревожности клиента по поводу задержек или качества.',
      },
    ],
  },
  {
    id: 'offer',
    label: 'КП / Скидки',
    icon: FileText,
    questions: [
      {
        label: 'Скидка 3%',
        prompt: 'Рассчитай условия покупки и платежи при скидке 3% на эту квартиру.',
      },
      {
        label: 'Скидка 5%',
        prompt: 'Рассчитай коммерческое предложение со скидкой 5% и покажи выгоду клиента.',
      },
      {
        label: 'Ипотека vs рассрочка',
        prompt: 'Подготовь сравнение покупки: 100% оплата, семейная ипотека и рассрочка от застройщика.',
      },
    ],
  },
  {
    id: 'general',
    label: 'Общий вопрос',
    icon: Sparkles,
    questions: [
      {
        label: 'Регламент сделки',
        prompt: 'Напомни этапы оформления сделки: бронирование, одобрение ипотеки, подписание ДДУ и открытие эскроу-счёта.',
      },
      {
        label: 'Плюсы инфраструктуры',
        prompt: 'Перечисли основные преимущества инфраструктуры, транспорта и окружения для этого жилого комплекса.',
      },
    ],
  },
];

const modeAgentLabels: Record<string, string> = {
  general: 'Общий',
  negotiation: 'Переговоры',
  analytics: 'Аналитика',
  offer: 'КП',
};

const createMessageId = () =>
  typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(16).slice(2)}`;

const formatAiError = (err: unknown, defaultMessage: string) => {
  const status =
    typeof err === 'object' && err !== null && 'status' in err
      ? Number((err as { status?: number }).status)
      : undefined;

  if (status === 401) return 'Сессия истекла. Войдите снова.';
  if (status === 403) return 'Нет доступа к этому диалогу.';
  if (status === 404) return 'Данные диалога не найдены.';
  if (status === 409) return 'Операция недоступна в текущем состоянии.';
  if (status === 502) return 'AI не смог обработать запрос.';
  if (status === 503 || status === 504) return 'AI-сервис временно недоступен.';
  return err instanceof Error && err.message ? err.message : defaultMessage;
};

const formatInlineText = (text: string) =>
  text
    .split(/(\*\*[^*]+\*\*)/g)
    .filter(Boolean)
    .map((part, index) =>
      part.startsWith('**') && part.endsWith('**') ? (
        <strong key={`${part}-${index}`} className="font-extrabold text-zinc-900">
          {part.slice(2, -2)}
        </strong>
      ) : (
        <React.Fragment key={`${part}-${index}`}>{part}</React.Fragment>
      ),
    );

const FormattedAIText: React.FC<{ text: string }> = ({ text }) => (
  <div className="[overflow-wrap:anywhere] space-y-1.5">
    {text.split('\n').map((line, index) => {
      if (!line.trim()) return <div key={`space-${index}`} className="h-1.5" aria-hidden="true" />;
      if (line.startsWith('### ')) {
        return (
          <h4 key={`heading-${index}`} className="mb-1 mt-2 text-xs font-black text-zinc-900 first:mt-0 uppercase tracking-wide">
            {formatInlineText(line.slice(4))}
          </h4>
        );
      }
      if (line.startsWith('## ')) {
        return (
          <h3 key={`heading-${index}`} className="mb-1.5 mt-2.5 text-sm font-black text-zinc-900 first:mt-0">
            {formatInlineText(line.slice(3))}
          </h3>
        );
      }
      if (line.startsWith('- ') || line.startsWith('* ')) {
        return (
          <div key={`list-${index}`} className="my-0.5 flex items-start gap-2">
            <span className="mt-[0.55em] h-1.5 w-1.5 shrink-0 rounded-full bg-amber-500" />
            <p className="min-w-0 flex-1">{formatInlineText(line.slice(2))}</p>
          </div>
        );
      }
      return (
        <p key={`paragraph-${index}`} className="my-0.5 first:mt-0 last:mb-0">
          {formatInlineText(line)}
        </p>
      );
    })}
  </div>
);

export const ManagerAIPanel: React.FC<ManagerAIPanelProps> = ({
  dealId,
  sessionId,
  clientName,
  apartmentNumber,
  basePrice,
  chatMessages,
  onApplyReplyText,
  onClose,
}) => {
  const [messages, setMessages] = useState<AIMessage[]>([]);
  const [input, setInput] = useState('');
  const [discount, setDiscount] = useState('');
  const [activeCategory, setActiveCategory] = useState<ChatMode | null>(null);
  const [loadingMode, setLoadingMode] = useState<ChatMode | null>(null);
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null);
  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(false);
  const historyRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const chipsScrollRef = useRef<HTMLDivElement>(null);
  const activeSessionIdRef = useRef(sessionId);

  activeSessionIdRef.current = sessionId;

  const updateScrollState = () => {
    const el = chipsScrollRef.current;
    if (!el) {
      setCanScrollLeft(false);
      setCanScrollRight(false);
      return;
    }
    const { scrollLeft, scrollWidth, clientWidth } = el;
    setCanScrollLeft(scrollLeft > 2);
    setCanScrollRight(scrollLeft + clientWidth < scrollWidth - 2);
  };

  const scrollChips = (direction: 'left' | 'right') => {
    if (chipsScrollRef.current) {
      const scrollAmount = direction === 'left' ? -200 : 200;
      chipsScrollRef.current.scrollBy({ left: scrollAmount, behavior: 'smooth' });
    }
  };

  useEffect(() => {
    const el = chipsScrollRef.current;
    if (!el) return;

    const timer = setTimeout(updateScrollState, 50);
    el.addEventListener('scroll', updateScrollState, { passive: true });
    window.addEventListener('resize', updateScrollState);

    return () => {
      clearTimeout(timer);
      el.removeEventListener('scroll', updateScrollState);
      window.removeEventListener('resize', updateScrollState);
    };
  }, [activeCategory]);

  useEffect(() => {
    setMessages([]);
    setInput('');
    setDiscount('');
    setActiveCategory(null);
    setLoadingMode(null);
    setCopiedMessageId(null);
  }, [sessionId]);

  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      const history = historyRef.current;
      if (history) history.scrollTo({ top: history.scrollHeight, behavior: 'smooth' });
    });
    return () => cancelAnimationFrame(frame);
  }, [messages, loadingMode]);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, 84)}px`;
  }, [input]);

  const appendMessage = (message: Omit<AIMessage, 'id' | 'createdAt'>) => {
    setMessages((current) => [
      ...current,
      { ...message, id: createMessageId(), createdAt: new Date().toISOString() },
    ]);
  };

  const appendError = (targetMode: ChatMode, error: unknown, fallback: string) => {
    appendMessage({
      role: 'system',
      text: formatAiError(error, fallback),
      mode: targetMode,
      error: true,
    });
  };

  const buildContextDescription = () => {
    const parts: string[] = [];
    if (clientName) parts.push(`Клиент: ${clientName}`);
    if (apartmentNumber) parts.push(`Целевая квартира №${apartmentNumber}`);
    if (basePrice) parts.push(`Стоимость объекта: ${formatPrice(basePrice)}`);
    if (dealId) parts.push(`Сделка #${dealId}`);
    return parts.join(', ');
  };

  const formatChatTranscript = () => {
    if (!chatMessages || chatMessages.length === 0) return '';
    const formatted = chatMessages
      .map((m) => {
        const sender =
          m.sender_type === 'client'
            ? 'Клиент'
            : m.sender_type === 'manager'
            ? 'Менеджер'
            : m.sender_type === 'ai'
            ? 'ИИ'
            : 'Система';
        return `${sender}: ${m.content}`;
      })
      .slice(-25)
      .join('\n');
    return `\n\n[История открытой переписки в диалоге]:\n${formatted}`;
  };

  const buildModePrompt = (targetMode: ChatMode, text: string) => {
    const ctx = buildContextDescription();
    const transcript = formatChatTranscript();
    const ctxPrefix = ctx ? `[Контекст диалога: ${ctx}] ` : '';

    if (targetMode === 'negotiation') {
      return `${ctxPrefix}Переговоры с клиентом по квартире. ${text}${transcript}`;
    }
    if (targetMode === 'risks') {
      return `${ctxPrefix}Анализ рисков застройки и строительства по объекту. ${text}${transcript}`;
    }
    if (targetMode === 'offer') {
      return `${ctxPrefix}Коммерческое предложение и расчёт скидки на квартиру. ${text}${transcript}`;
    }
    if (targetMode === 'analysis') {
      return `${ctxPrefix}Анализ текущей открытой переписки: ${text}${transcript}`;
    }
    if (targetMode === 'reply') {
      return `${ctxPrefix}Подготовка черновика ответа клиенту в чат: ${text}${transcript}`;
    }
    return `${ctxPrefix}${text}${transcript}`;
  };

  const handleChatSend = async (messageText?: string, explicitMode?: ChatMode) => {
    const text = (messageText ?? input).trim();
    if (!text || loadingMode) return;

    const requestedMode = explicitMode || activeCategory || 'general';
    const requestedSessionId = sessionId;
    appendMessage({ role: 'user', text, mode: requestedMode });
    setInput('');
    setLoadingMode(requestedMode);

    try {
      const response = await aiApi.askAssistant({
        message: buildModePrompt(requestedMode, text),
        session_id: requestedSessionId,
        deal_id: dealId,
      });
      if (activeSessionIdRef.current !== requestedSessionId) return;
      appendMessage({
        role: 'assistant',
        text: response.message,
        agent: response.agent,
        intent: response.intent,
        mode: requestedMode,
      });
    } catch (error) {
      if (activeSessionIdRef.current === requestedSessionId) {
        appendError(requestedMode, error, 'Не удалось получить ответ AI.');
      }
    } finally {
      if (activeSessionIdRef.current === requestedSessionId) setLoadingMode(null);
    }
  };

  const handleOfferCreate = () => {
    const value = discount.trim().replace(',', '.');
    const prompt = value
      ? `Сформируй коммерческое предложение со скидкой ${value}% на целевую квартиру.`
      : 'Сформируй коммерческое предложение на целевую квартиру.';
    void handleChatSend(prompt, 'offer');
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      void handleChatSend();
    }
  };

  const copyReply = async (message: AIMessage) => {
    try {
      await navigator.clipboard.writeText(message.replyAssist?.suggested_reply ?? message.text);
      setCopiedMessageId(message.id);
      window.setTimeout(() => setCopiedMessageId(null), 1500);
    } catch {
      setCopiedMessageId(null);
    }
  };

  const currentCategory = CATEGORIES.find((c) => c.id === activeCategory) ?? null;

  return (
    <section className="flex h-full min-h-0 flex-col overflow-hidden bg-white">

      <header className="shrink-0 border-b-2 border-zinc-900 bg-[#FAF8F2] px-4 py-2.5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-zinc-900 text-amber-400 border-2 border-zinc-900 shadow-xs">
              <Sparkles className="h-3.5 w-3.5" />
            </span>
            <h3 className="text-xs font-black uppercase tracking-wider text-zinc-900">AI-Ассистент</h3>
          </div>

          <div>
            {onClose && (
              <button
                type="button"
                onClick={onClose}
                className="p-1.5 rounded-lg bg-white border-2 border-zinc-900 hover:bg-zinc-100 text-zinc-800 cursor-pointer shadow-xs transition-colors flex items-center justify-center"
                title="Закрыть AI-ассистент"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      </header>

      <div ref={historyRef} className="min-h-0 flex-1 space-y-3.5 overflow-y-auto bg-[#FAF8F2]/40 px-4 py-3 sm:px-5" aria-live="polite">
        {messages.length === 0 && (
          <div className="flex h-full min-h-36 flex-col items-center justify-center px-6 text-center">
            <span className="mb-2.5 flex h-10 w-10 items-center justify-center rounded-2xl bg-white text-amber-500 border-2 border-zinc-900 shadow-xs">
              <Sparkles className="h-5 w-5" />
            </span>
            <p className="text-xs font-black uppercase tracking-wider text-zinc-900">Рабочий помощник диалога</p>
            <p className="mt-1 max-w-md text-xs font-medium text-zinc-500">
              Выберите категорию внизу для перехода к быстрым сценариям или задайте вопрос в поле ввода.
            </p>
          </div>
        )}

        {messages.map((message) => (
          <MessageItem
            key={message.id}
            message={message}
            copied={copiedMessageId === message.id}
            onApplyReplyText={onApplyReplyText}
            onCopy={() => void copyReply(message)}
          />
        ))}

        {loadingMode && (
          <div className="flex justify-start">
            <div>
              <div className="mb-1 flex items-center gap-1.5 px-1 text-[10px] font-extrabold text-zinc-700">
                <Sparkles className="h-3 w-3 text-amber-500" />
                AI-ассистент анализирует диалог и формирует ответ...
              </div>
              <div className="flex h-8 items-center gap-1.5 rounded-2xl rounded-tl-xs border-2 border-zinc-900 bg-[#FFFDF8] px-4 shadow-xs" aria-label="AI думает">
                <span className="h-2 w-2 animate-bounce rounded-full bg-zinc-900 [animation-delay:-0.3s]" />
                <span className="h-2 w-2 animate-bounce rounded-full bg-amber-500 [animation-delay:-0.15s]" />
                <span className="h-2 w-2 animate-bounce rounded-full bg-zinc-900" />
              </div>
            </div>
          </div>
        )}
      </div>

      <footer className="shrink-0 border-t-2 border-zinc-900 bg-white p-3 shadow-xs">

        <div className="group/chips relative mb-2.5 flex items-center">
          {canScrollLeft && (
            <button
              type="button"
              onClick={() => scrollChips('left')}
              className="absolute left-0 z-10 flex h-full items-center justify-center pl-0.5 pr-2 text-zinc-800 hover:text-zinc-950 opacity-0 group-hover/chips:opacity-100 transition-opacity cursor-pointer bg-gradient-to-r from-white/90 via-white/50 to-transparent"
              title="Прокрутить влево"
              aria-label="Прокрутить влево"
            >
              <ChevronLeft className="h-5 w-5 drop-shadow-xs" />
            </button>
          )}

          <div
            ref={chipsScrollRef}
            className="w-full flex items-center gap-1.5 overflow-x-auto scroll-smooth py-0.5 no-scrollbar"
            style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
          >
            {activeCategory === null ? (

              CATEGORIES.map((cat) => {
                const Icon = cat.icon;
                return (
                  <button
                    key={cat.id}
                    type="button"
                    disabled={Boolean(loadingMode)}
                    onClick={() => setActiveCategory(cat.id)}
                    className="flex shrink-0 items-center gap-1.5 px-3 py-1.5 rounded-xl border-2 border-zinc-900 bg-[#FAF8F2] hover:bg-amber-100 text-zinc-900 text-xs font-bold transition-all cursor-pointer shadow-xs disabled:opacity-40 whitespace-nowrap"
                  >
                    <Icon className="w-3.5 h-3.5 text-zinc-800" />
                    <span>{cat.label}</span>
                  </button>
                );
              })
            ) : (

              <>
                <button
                  type="button"
                  onClick={() => setActiveCategory(null)}
                  className="flex shrink-0 items-center gap-1 px-2.5 py-1.5 rounded-xl border-2 border-zinc-900 bg-zinc-900 hover:bg-zinc-800 text-white text-xs font-bold transition-all cursor-pointer shadow-xs whitespace-nowrap"
                  title="Вернуться к категориям"
                >
                  <ArrowLeft className="w-3.5 h-3.5" />
                  <span>Назад</span>
                </button>

                {currentCategory?.questions.map((q, idx) => (
                  <button
                    key={idx}
                    type="button"
                    disabled={Boolean(loadingMode)}
                    onClick={() => void handleChatSend(q.prompt, currentCategory.id)}
                    className="shrink-0 px-3 py-1.5 rounded-xl border-2 border-zinc-900 bg-white hover:bg-amber-100 text-zinc-900 text-xs font-bold transition-all cursor-pointer shadow-xs disabled:opacity-40 whitespace-nowrap"
                  >
                    {q.label}
                  </button>
                ))}
              </>
            )}
          </div>

          {canScrollRight && (
            <button
              type="button"
              onClick={() => scrollChips('right')}
              className="absolute right-0 z-10 flex h-full items-center justify-center pr-0.5 pl-2 text-zinc-800 hover:text-zinc-950 opacity-0 group-hover/chips:opacity-100 transition-opacity cursor-pointer bg-gradient-to-l from-white/90 via-white/50 to-transparent"
              title="Прокрутить вправо"
              aria-label="Прокрутить вправо"
            >
              <ChevronRight className="h-5 w-5 drop-shadow-xs" />
            </button>
          )}
        </div>

        {activeCategory === 'offer' && (
          <div className="mb-2 flex flex-wrap items-center gap-2 rounded-xl bg-[#FAF8F2] border-2 border-zinc-900 p-2">
            {basePrice !== undefined && (
              <span className="mr-auto text-[11px] font-bold text-zinc-600">
                Базовая цена: <strong className="text-zinc-900 font-extrabold">{formatPrice(basePrice)}</strong>
              </span>
            )}
            <label htmlFor="ai-offer-discount" className="text-[11px] font-extrabold text-zinc-900">Скидка:</label>
            <div className="relative w-20">
              <input
                id="ai-offer-discount"
                type="text"
                inputMode="decimal"
                value={discount}
                onChange={(event) => setDiscount(event.target.value)}
                disabled={Boolean(loadingMode)}
                placeholder="3"
                className="w-full rounded-lg border-2 border-zinc-900 bg-white px-2 py-1 pr-5 text-xs font-bold text-zinc-900 outline-none focus:bg-[#FFFDF8]"
              />
              <span className="absolute right-2 top-1 text-xs font-bold text-zinc-500">%</span>
            </div>
            <button
              type="button"
              onClick={handleOfferCreate}
              disabled={Boolean(loadingMode)}
              className="rounded-lg bg-zinc-900 px-3 py-1.5 text-[10px] font-extrabold text-white hover:bg-zinc-800 border-2 border-zinc-900 cursor-pointer disabled:opacity-40"
            >
              Сформировать КП
            </button>
          </div>
        )}

        <div className="flex items-end gap-2 rounded-xl border-2 border-zinc-900 bg-[#FAF8F2] p-1.5 focus-within:bg-white">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(event) => setInput(event.target.value)}
            onKeyDown={handleKeyDown}
            rows={1}
            placeholder="Задайте вопрос AI по открытой переписке или объекту..."
            disabled={Boolean(loadingMode)}
            className="min-h-[36px] max-h-20 flex-1 resize-none overflow-y-auto bg-transparent px-2 py-1 text-xs font-bold text-zinc-900 outline-none placeholder:text-zinc-400 disabled:opacity-60"
          />
          <button
            type="button"
            onClick={() => void handleChatSend()}
            disabled={Boolean(loadingMode) || !input.trim()}
            className="px-5 py-2.5 bg-zinc-900 hover:bg-zinc-800 text-white rounded-xl text-xs font-bold flex items-center gap-1.5 flex-shrink-0 transition-colors disabled:opacity-40 border-2 border-zinc-900 cursor-pointer"
            title="Отправить сообщение"
          >
            {loadingMode ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Send className="w-3.5 h-3.5" />}
            <span>Отправить</span>
          </button>
        </div>
      </footer>
    </section>
  );
};

const MessageItem: React.FC<{
  message: AIMessage;
  copied: boolean;
  onApplyReplyText?: (text: string) => void;
  onCopy: () => void;
}> = ({ message, copied, onApplyReplyText, onCopy }) => {
  if (message.role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[80%]">
          <div className="rounded-2xl rounded-br-xs border-2 border-zinc-900 bg-zinc-900 px-3.5 py-2.5 text-xs leading-relaxed text-white shadow-xs font-medium">
            <p className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]">{message.text}</p>
          </div>
          <MessageMeta message={message} align="right" />
        </div>
      </div>
    );
  }

  if (message.analysis) return <AnalysisMessage message={message} result={message.analysis} />;
  if (message.replyAssist) {
    return (
      <ReplyMessage
        message={message}
        result={message.replyAssist}
        copied={copied}
        onApplyReplyText={onApplyReplyText}
        onCopy={onCopy}
      />
    );
  }

  if (message.role === 'system') {
    return (
      <div className={`mx-auto flex max-w-[92%] items-start gap-2 rounded-xl border-2 px-3 py-2.5 text-xs font-bold ${
        message.error ? 'border-rose-900 bg-rose-50 text-rose-800' : 'border-zinc-900 bg-white text-zinc-700 shadow-xs'
      }`}>
        <AlertCircle className={`mt-0.5 h-4 w-4 shrink-0 ${message.error ? 'text-rose-600' : 'text-zinc-500'}`} />
        <div className="flex-1">
          <span>{message.text}</span>
          <MessageMeta message={message} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex justify-start">
      <div className="min-w-0 max-w-[88%]">
        <div className="mb-1 flex items-center gap-1.5 px-1">
          <span className="flex h-5 w-5 items-center justify-center rounded-lg bg-amber-400 text-zinc-900 border border-zinc-900">
            <Sparkles className="h-3 w-3" />
          </span>
          <span className="text-[10px] font-black uppercase tracking-wider text-zinc-900">ИИ-Помощник</span>
          {message.agent && (
            <span className="rounded-md border border-zinc-900 bg-[#FAF8F2] px-1.5 py-0.2 text-[9px] font-extrabold text-zinc-800">
              {modeAgentLabels[message.agent] ?? message.agent}
            </span>
          )}
        </div>
        <article className="rounded-2xl rounded-tl-xs border-2 border-zinc-900 bg-[#FFFDF8] px-3.5 py-3 text-xs leading-relaxed text-zinc-900 shadow-xs font-medium">
          <FormattedAIText text={message.text} />

          <div className="mt-2.5 flex items-center justify-between border-t border-zinc-200 pt-2">
            <MessageMeta message={message} />
            {onApplyReplyText && (
              <button
                type="button"
                onClick={() => onApplyReplyText(message.text)}
                className="flex items-center gap-1 rounded-lg border-2 border-zinc-900 bg-emerald-600 px-2.5 py-1 text-[10px] font-extrabold text-white transition hover:bg-emerald-700 cursor-pointer shadow-xs"
                title="Вставить этот текст в поле ввода диалога с клиентом"
              >
                <Check className="h-3 w-3" />
                Вставить в ответ
              </button>
            )}
          </div>
        </article>
      </div>
    </div>
  );
};

const MessageMeta: React.FC<{ message: AIMessage; align?: 'left' | 'right' }> = ({
  message,
  align = 'left',
}) => (
  <div className={`mt-0.5 flex items-center gap-1.5 px-1 text-[9px] font-bold text-zinc-400 ${align === 'right' ? 'justify-end' : ''}`}>
    <time dateTime={message.createdAt}>
      {new Date(message.createdAt).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}
    </time>
    {message.intent && <span>• {message.intent}</span>}
  </div>
);

const AnalysisMessage: React.FC<{ message: AIMessage; result: DialogAnalysisResult }> = ({
  message,
  result,
}) => {
  const facts = result.analysis;
  const legacyNeeds = result.client_needs;
  const importantFactors = facts?.important_factors ?? [];
  const objections = facts?.objections ?? result.objections ?? [];
  const summary = facts?.summary ?? result.summary;

  return (
    <div className="flex justify-start">
      <div className="min-w-0 max-w-[92%] space-y-2">
        <div className="mb-1 flex items-center gap-1.5 px-1">
          <span className="flex h-5 w-5 items-center justify-center rounded-lg bg-amber-400 text-zinc-900 border border-zinc-900">
            <UserRoundSearch className="h-3 w-3" />
          </span>
          <span className="text-[10px] font-black uppercase tracking-wider text-zinc-900">Анализ профиля клиента</span>
        </div>

        <article className="rounded-2xl rounded-tl-xs border-2 border-zinc-900 bg-[#FFFDF8] p-3.5 text-xs text-zinc-900 shadow-xs space-y-3 font-medium">
          {summary && (
            <div>
              <h5 className="text-[10px] font-black uppercase tracking-wider text-zinc-500 mb-1">Сводка диалога</h5>
              <p className="rounded-xl border border-zinc-900 bg-white p-2.5 font-bold leading-relaxed">{summary}</p>
            </div>
          )}

          {facts && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px]">
              <div className="rounded-xl border border-zinc-900 bg-white p-2">
                <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Локация / Район</span>
                <span className="font-extrabold text-zinc-900">{facts.preferred_district || 'Не указан'}</span>
              </div>
              <div className="rounded-xl border border-zinc-900 bg-white p-2">
                <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Комнатность</span>
                <span className="font-extrabold text-zinc-900">{facts.rooms ? `${facts.rooms}-комнатная` : 'Любая'}</span>
              </div>
              <div className="rounded-xl border border-zinc-900 bg-white p-2">
                <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Срок покупки</span>
                <span className="font-extrabold text-zinc-900">{facts.purchase_timeline || 'В процессе'}</span>
              </div>
              <div className="rounded-xl border border-zinc-900 bg-white p-2">
                <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Бюджет</span>
                <span className="font-extrabold text-zinc-900">
                  {facts.budget_max ? formatPrice(facts.budget_max) : 'Не зафиксирован'}
                </span>
              </div>
            </div>
          )}

          {legacyNeeds && !facts && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px]">
              {legacyNeeds.district && (
                <div className="rounded-xl border border-zinc-900 bg-white p-2">
                  <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Район</span>
                  <span className="font-extrabold text-zinc-900">{legacyNeeds.district}</span>
                </div>
              )}
              {legacyNeeds.rooms && (
                <div className="rounded-xl border border-zinc-900 bg-white p-2">
                  <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Комнат</span>
                  <span className="font-extrabold text-zinc-900">{legacyNeeds.rooms}</span>
                </div>
              )}
              {legacyNeeds.budget_max && (
                <div className="rounded-xl border border-zinc-900 bg-white p-2">
                  <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Бюджет</span>
                  <span className="font-extrabold text-zinc-900">{formatPrice(legacyNeeds.budget_max)}</span>
                </div>
              )}
              {legacyNeeds.preferred_finishing && (
                <div className="rounded-xl border border-zinc-900 bg-white p-2">
                  <span className="text-[9px] font-black uppercase tracking-wider text-zinc-500 block mb-0.5">Отделка</span>
                  <span className="font-extrabold text-zinc-900">{legacyNeeds.preferred_finishing}</span>
                </div>
              )}
            </div>
          )}

          {importantFactors.length > 0 && (
            <div>
              <h5 className="text-[10px] font-black uppercase tracking-wider text-zinc-500 mb-1.5">Ключевые факторы</h5>
              <div className="flex flex-wrap gap-1">
                {importantFactors.map((factor, idx) => (
                  <span key={idx} className="rounded-lg border border-zinc-900 bg-amber-200 px-2 py-0.5 text-[10px] font-extrabold text-zinc-900">
                    {factor}
                  </span>
                ))}
              </div>
            </div>
          )}

          {objections.length > 0 && (
            <div>
              <h5 className="text-[10px] font-black uppercase tracking-wider text-zinc-500 mb-1.5">Возражения и сомнения</h5>
              <div className="space-y-1">
                {objections.map((obj, idx) => (
                  <div key={idx} className="rounded-lg border border-rose-900 bg-rose-50 px-2.5 py-1 text-[11px] font-bold text-rose-900 flex items-center gap-1.5">
                    <span className="w-1.5 h-1.5 rounded-full bg-rose-600 shrink-0" />
                    <span>{String(obj)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <MessageMeta message={message} />
        </article>
      </div>
    </div>
  );
};

const ReplyMessage: React.FC<{
  message: AIMessage;
  result: ReplyAssistResult;
  copied: boolean;
  onApplyReplyText?: (text: string) => void;
  onCopy: () => void;
}> = ({ message, result, copied, onApplyReplyText, onCopy }) => {
  const replyText = result.suggested_reply || message.text;

  return (
    <div className="flex justify-start">
      <div className="min-w-0 max-w-[92%] space-y-2">
        <div className="mb-1 flex items-center justify-between px-1">
          <div className="flex items-center gap-1.5">
            <span className="flex h-5 w-5 items-center justify-center rounded-lg bg-amber-400 text-zinc-900 border border-zinc-900">
              <Bot className="h-3 w-3" />
            </span>
            <span className="text-[10px] font-black uppercase tracking-wider text-zinc-900">Готовый черновик ответа</span>
          </div>
          {result.analysis?.intent && (
            <span className="rounded-md border border-zinc-900 bg-[#FAF8F2] px-1.5 py-0.2 text-[9px] font-extrabold text-zinc-800">
              {result.analysis.intent}
            </span>
          )}
        </div>

        <article className="rounded-2xl rounded-tl-xs border-2 border-zinc-900 bg-[#FFFDF8] p-3.5 text-xs leading-relaxed text-zinc-900 shadow-xs font-medium">
          <div className="rounded-xl border-2 border-zinc-900 bg-white p-3 font-bold text-zinc-900">
            <p className="whitespace-pre-wrap">{replyText}</p>
          </div>

          <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-zinc-200 pt-2.5">
            <MessageMeta message={message} />
            <div className="flex items-center gap-1.5">
              <button
                type="button"
                onClick={onCopy}
                className="flex items-center gap-1 rounded-lg border-2 border-zinc-900 bg-white px-2.5 py-1 text-[10px] font-extrabold text-zinc-900 transition hover:bg-[#FAF8F2] cursor-pointer shadow-xs"
              >
                {copied ? <Check className="h-3 w-3 text-emerald-600" /> : <Copy className="h-3 w-3" />}
                {copied ? 'Скопировано' : 'Копировать'}
              </button>

              {onApplyReplyText && (
                <button
                  type="button"
                  onClick={() => onApplyReplyText(replyText)}
                  className="flex items-center gap-1 rounded-lg border-2 border-zinc-900 bg-emerald-600 px-3 py-1 text-[10px] font-extrabold text-white transition hover:bg-emerald-700 cursor-pointer shadow-xs"
                >
                  <Check className="h-3 w-3" />
                  Вставить в поле ввода
                </button>
              )}
            </div>
          </div>
        </article>
      </div>
    </div>
  );
};

const ContextBadge: React.FC<{
  icon: React.ComponentType<{ className?: string }>;
  text: string;
  highlighted?: boolean;
}> = ({ icon: Icon, text, highlighted }) => (
  <span
    className={`inline-flex items-center gap-1 rounded-xl px-2 py-0.5 text-[10px] font-extrabold border-2 border-zinc-900 ${
      highlighted
        ? 'bg-amber-400 text-zinc-900 shadow-xs'
        : 'bg-white text-zinc-900 shadow-xs'
    }`}
  >
    <Icon className="h-3 w-3" />
    <span className="truncate max-w-[130px]">{text}</span>
  </span>
);
