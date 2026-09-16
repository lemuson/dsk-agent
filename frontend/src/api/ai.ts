import { api } from './client';
import { DialogAnalysisResult, ReplyAssistResult, AIChatResponse } from '../types';

export const aiApi = {

  analyzeDialog: async (dealId: number): Promise<DialogAnalysisResult> => {
    return api.post<DialogAnalysisResult>('/ai/dialog/analyze', { deal_id: dealId }, true);
  },

  replyAssist: async (dealId: number, selectedText?: string): Promise<ReplyAssistResult> => {
    return api.post<ReplyAssistResult>(
      '/ai/dialog/reply-assist',
      { deal_id: dealId, selected_text: selectedText },
      true
    );
  },

  askAssistant: async (
    payload: {
      session_id: number;
      deal_id?: number;
      message: string;
      parking_unit_id?: number;
      storage_unit_id?: number;
    },
    isStaff = true
  ): Promise<AIChatResponse> => {
    return api.post<AIChatResponse>('/ai/chat', payload, isStaff);
  },
};
