import React, { useEffect, useMemo, useState } from 'react';
import { Bot, Loader2, Sparkles, X } from 'lucide-react';
import { chatsApi } from '../../api/chats';
import { dealsApi } from '../../api/deals';
import { AssistantChatContext } from '../../context/AssistantContext';
import { ChatSession, Deal } from '../../types';
import { ManagerAIPanel } from './ManagerAIPanel';

interface GlobalAIAssistantModalProps {
  isOpen: boolean;
  onClose: () => void;
  activeContext: AssistantChatContext | null;
  onApplyReply: (text: string, sessionId: number) => boolean;
}

export const GlobalAIAssistantModal: React.FC<GlobalAIAssistantModalProps> = ({
  isOpen,
  onClose,
  activeContext,
  onApplyReply,
}) => {
  const [deals, setDeals] = useState<Deal[]>([]);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [selectedSessionId, setSelectedSessionId] = useState<number | null>(null);
  const [loadingData, setLoadingData] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setLoadingData(true);
    Promise.all([
      dealsApi.getAllDeals().catch(() => []),
      chatsApi.getAllSessions().catch(() => []),
    ])
      .then(([dealsList, sessionsList]) => {
        setDeals(dealsList);
        setSessions(sessionsList);
      })
      .finally(() => setLoadingData(false));
  }, [isOpen]);

  const contexts = useMemo(() => {
    const result: AssistantChatContext[] = [];
    if (activeContext) result.push(activeContext);

    for (const deal of deals) {
      if (!deal.id_chat_session || result.some((item) => item.sessionId === deal.id_chat_session)) continue;
      result.push({
        sessionId: deal.id_chat_session,
        dealId: deal.id,
        clientName: deal.user_name,
        apartmentNumber: deal.apartment_number,
        basePrice: deal.base_price,
      });
    }

    for (const sess of sessions) {
      if (result.some((item) => item.sessionId === sess.id)) continue;
      result.push({
        sessionId: sess.id,
        clientName: sess.user_name || sess.guest_name,
      });
    }

    return result;
  }, [activeContext, deals, sessions]);

  useEffect(() => {
    if (!isOpen) return;
    const preferredSessionId = activeContext?.sessionId ?? contexts[0]?.sessionId ?? null;
    setSelectedSessionId((current) =>
      current && contexts.some((item) => item.sessionId === current) ? current : preferredSessionId,
    );
  }, [activeContext?.sessionId, contexts, isOpen]);

  if (!isOpen) return null;

  const selectedContext = contexts.find((item) => item.sessionId === selectedSessionId) ?? null;
  const canInsertIntoActiveChat = Boolean(
    selectedContext && activeContext?.sessionId === selectedContext.sessionId,
  );
  const handleApplyReply = canInsertIntoActiveChat && selectedContext
    ? (text: string) => {
        if (onApplyReply(text, selectedContext.sessionId)) onClose();
      }
    : undefined;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/60 p-3 backdrop-blur-sm sm:p-5">
      <div className="flex h-[min(900px,94vh)] w-full max-w-5xl flex-col overflow-hidden rounded-3xl border-2 border-zinc-900 bg-[#FAF8F2] shadow-2xl">
        <header className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b-2 border-zinc-900 bg-white px-4 py-3 sm:px-5">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-2xl bg-zinc-900 text-amber-400 border-2 border-zinc-900 shadow-xs">
              <Bot className="h-5 w-5" />
            </span>
            <div>
              <div className="flex items-center gap-1.5">
                <h2 className="text-sm font-black uppercase tracking-wider text-zinc-900">AI-Ассистент ДСК</h2>
                <span className="rounded-full bg-amber-400 px-2 py-0.2 text-[9px] font-black text-zinc-900 border border-zinc-900">
                  Менеджер
                </span>
              </div>
              <p className="text-[11px] font-bold text-zinc-500">Помощник по диалогам, объектам и переговорам</p>
            </div>
          </div>

          <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
            {loadingData && <Loader2 className="h-4 w-4 animate-spin text-zinc-600" />}
            <label htmlFor="assistant-context" className="hidden text-[10px] font-extrabold uppercase tracking-wider text-zinc-500 sm:block">
              Контекст диалога:
            </label>
            <select
              id="assistant-context"
              value={selectedSessionId ?? ''}
              onChange={(event) => setSelectedSessionId(Number(event.target.value))}
              disabled={contexts.length === 0}
              className="min-w-0 max-w-xs rounded-xl border-2 border-zinc-900 bg-white px-3 py-1.5 text-xs font-bold text-zinc-900 outline-none focus:bg-[#FAF8F2] disabled:opacity-50"
            >
              {contexts.length === 0 && <option value="">Нет доступных диалогов</option>}
              {contexts.map((context) => (
                <option key={context.sessionId} value={context.sessionId}>
                  {context.clientName || `Диалог #${context.sessionId}`}
                  {context.apartmentNumber ? ` (Кв. №${context.apartmentNumber})` : ''}
                  {context.dealId ? ` · Сделка #${context.dealId}` : ' · Лид'}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={onClose}
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border-2 border-zinc-900 bg-white text-zinc-900 transition hover:bg-zinc-100 cursor-pointer shadow-xs"
              aria-label="Закрыть AI-ассистент"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </header>

        <main className="min-h-0 flex-1 p-3 sm:p-4">
          {selectedContext ? (
            <ManagerAIPanel
              sessionId={selectedContext.sessionId}
              dealId={selectedContext.dealId}
              clientName={selectedContext.clientName}
              apartmentNumber={selectedContext.apartmentNumber}
              basePrice={selectedContext.basePrice}
              onApplyReplyText={handleApplyReply}
            />
          ) : (
            <div className="flex h-full items-center justify-center rounded-2xl border-2 border-dashed border-zinc-400 bg-white px-6 text-center">
              <div className="max-w-md">
                <Bot className="mx-auto h-8 w-8 text-zinc-400" />
                <p className="mt-3 text-sm font-extrabold text-zinc-900">Нет доступных диалогов</p>
                <p className="mt-1 text-xs leading-relaxed text-zinc-500 font-medium">
                  Выберите диалог в рабочем кабинете, чтобы AI-ассистент мог анализировать историю сообщений и целевой объект.
                </p>
              </div>
            </div>
          )}
        </main>
      </div>
    </div>
  );
};
