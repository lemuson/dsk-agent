import { api } from './client';
import { DiscountPolicy, Role, User } from '../types';

export interface CreateDiscountPolicyPayload {
  building_id: number;
  role: Role;
  max_discount_percent: string;
  valid_from: string;
}

export const discountsApi = {

  getPoliciesByBuilding: async (buildingId: number): Promise<DiscountPolicy[]> => {
    return api.get<DiscountPolicy[]>(`/buildings/${buildingId}/discount-policies`, true);
  },

  createPolicy: async (payload: CreateDiscountPolicyPayload): Promise<DiscountPolicy> => {
    return api.post<DiscountPolicy>('/discount-policies', payload, true);
  },
};

export const staffApi = {

  getUsers: async (): Promise<User[]> => {
    return api.get<User[]>('/users', true);
  },

  getUser: async (id: number): Promise<User> => {
    return api.get<User>(`/users/${id}`, true);
  },
};
