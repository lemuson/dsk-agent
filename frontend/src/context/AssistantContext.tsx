import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';

export interface AssistantChatContext {
  sessionId: number;
  dealId?: number;
  clientName?: string;
  apartmentNumber?: string;
  basePrice?: number;
}

interface AssistantContextValue {
  activeChat: AssistantChatContext | null;
  registerActiveChat: (context: AssistantChatContext, applyReply: (text: string) => void) => void;
  clearActiveChat: (sessionId: number) => void;
  applyReply: (text: string, sessionId: number) => boolean;
}

const AssistantContext = createContext<AssistantContextValue | null>(null);

export const AssistantProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [activeChat, setActiveChat] = useState<AssistantChatContext | null>(null);
  const activeChatRef = useRef<AssistantChatContext | null>(null);
  const applyReplyRef = useRef<((text: string) => void) | null>(null);

  const registerActiveChat = useCallback((context: AssistantChatContext, applyReply: (text: string) => void) => {
    activeChatRef.current = context;
    applyReplyRef.current = applyReply;
    setActiveChat(context);
  }, []);

  const clearActiveChat = useCallback((sessionId: number) => {
    if (activeChatRef.current?.sessionId !== sessionId) return;
    activeChatRef.current = null;
    applyReplyRef.current = null;
    setActiveChat(null);
  }, []);

  const applyReply = useCallback((text: string, sessionId: number) => {
    if (activeChatRef.current?.sessionId !== sessionId || !applyReplyRef.current) return false;
    applyReplyRef.current(text);
    return true;
  }, []);

  const value = useMemo(
    () => ({ activeChat, registerActiveChat, clearActiveChat, applyReply }),
    [activeChat, registerActiveChat, clearActiveChat, applyReply],
  );

  return <AssistantContext.Provider value={value}>{children}</AssistantContext.Provider>;
};

export const useAssistant = () => {
  const context = useContext(AssistantContext);
  if (!context) throw new Error('useAssistant must be used inside AssistantProvider');
  return context;
};
