import { api } from './client';
import { Deal, Offer, DealStatus } from '../types';

export interface CreateDealPayload {
  id_user: number;
  id_apartment: number;
  id_chat_session?: number;
  percent_discount: number;
}

export interface CalculateOfferPayload {
  deal_id: number;
  discount_percent: string;
  parking_unit_id?: number;
  storage_unit_id?: number;
}

export interface CalculateOfferResult {
  deal_id: number;
  base_price: number;
  apartment_price: number;
  parking_unit_id?: number;
  parking_number?: string;
  parking_price: number;
  storage_unit_id?: number;
  storage_number?: string;
  storage_price: number;
  discount_percent: string;
  discount_amount: number;
  final_price: number;
  max_allowed_discount: string;
  requires_approval: boolean;
}

export const dealsApi = {

  getMyDeals: async (): Promise<Deal[]> => {
    return api.get<Deal[]>('/deals', false);
  },

  getDeal: async (id: number, isStaff = false): Promise<Deal> => {
    return api.get<Deal>(`/deals/${id}`, isStaff);
  },

  getAllDeals: async (): Promise<Deal[]> => {
    return api.get<Deal[]>('/deals', true);
  },

  createDeal: async (payload: CreateDealPayload): Promise<Deal> => {
    return api.post<Deal>('/deals', payload, true);
  },

  updateDealStatus: async (dealId: number, status: DealStatus, percentDiscount?: number, isStaff = true): Promise<Deal> => {
    return api.put<Deal>(`/deals/${dealId}/status`, { status, percent_discount: percentDiscount }, isStaff);
  },

  getMyOffers: async (): Promise<Offer[]> => {
    return api.get<Offer[]>('/offers', false);
  },

  getAllOffers: async (): Promise<Offer[]> => {
    return api.get<Offer[]>('/offers', true);
  },

  calculateOffer: async (payload: CalculateOfferPayload): Promise<CalculateOfferResult> => {
    return api.post<CalculateOfferResult>('/offers/calculate', payload, true);
  },

  createOffer: async (payload: {
    request_id: string;
    deal_id: number;
    discount_percent: string;
    generated_text: string;
    parking_unit_id?: number;
    storage_unit_id?: number;
  }): Promise<Offer> => {
    return api.post<Offer>('/offers', payload, true);
  },

  requestApproval: async (offerId: number): Promise<Offer> => {
    return api.post<Offer>(`/offers/${offerId}/approval-request`, {}, true);
  },

  approveOffer: async (offerId: number): Promise<Offer> => {
    return api.post<Offer>(`/offers/${offerId}/approve`, {}, true);
  },

  rejectOffer: async (offerId: number, reason: string): Promise<Offer> => {
    return api.post<Offer>(`/offers/${offerId}/reject`, { reason }, true);
  },

  downloadOfferPdf: async (offerId: number, isStaff = false): Promise<Blob> => {
    return api.get<Blob>(`/offers/${offerId}/pdf`, isStaff);
  },
};
