export type Role = 'user' | 'manager' | 'supervisor';

export interface User {
  id: number;
  name: string;
  email: string;
  role: Role;
  budget_max?: number;
  preferences?: Record<string, any>;
}

export type BuildingStatus = 'design' | 'construction' | 'completed' | 'suspended';
export type WallMaterial = 'panel' | 'monolith' | 'brick' | 'block';
export type FinishingType = 'rough' | 'white_box' | 'turnkey';
export type ApartmentStatus = 'free' | 'booked' | 'sold';

export interface ResidentialComplex {
  id: number;
  name: string;
  address: string;
  description: string;
}

export interface Building {
  id: number;
  residential_complex_id: number;
  address: string;
  district: string;
  latitude: number;
  longitude: number;
  floors_count: number;
  planned_date?: string;
  actual_date?: string;
  status: BuildingStatus;
  type_wall_material: WallMaterial;
  readiness_percent?: number;
  forecast_date?: string;
  delivery_shift_days?: number;
}

export interface Apartment {
  id: number;
  building_id: number;
  number: string;
  rooms: number;
  floor: number;
  area: number;
  price: number;
  type_finishing: FinishingType;
  status: ApartmentStatus;

  building?: Building;
  complex?: ResidentialComplex;
}

export interface ConstructionProgress {
  id: number;
  building_id: number;
  stage_name: 'excavation' | 'foundation' | 'frame' | 'roofing' | 'finishing';
  planned_start_date?: string;
  actual_start_date?: string;
  planned_end_date?: string;
  actual_end_date?: string;
  status: 'not_started' | 'in_progress' | 'completed' | 'delayed';
  completion_percentage?: number;
  delay_reason?: string;
  risk_level?: 'low' | 'medium' | 'high';
  delay_days?: number;
}

export type ChatSessionStatus = 'open' | 'in_progress' | 'pending_approval' | 'contract' | 'close';
export type SenderType = 'client' | 'manager' | 'system' | 'ai';

export interface ChatSession {
  id: number;
  id_user?: number;
  id_employee?: number;
  id_apartment?: number;
  guest_name?: string;
  guest_email?: string;
  guest_phone?: string;
  status: ChatSessionStatus;
  deleted_by_user?: boolean;
  created_at: string;
  updated_at: string;
  user_name?: string;
  employee_name?: string;

  apartment?: Apartment;
  user?: User;
}

export interface Message {
  id: number;
  id_chat_session: number;
  id_user?: number;
  sender_type: SenderType;
  content: string;
  is_read: boolean;
  sended_at: string;
  sender_name?: string;
}

export type DealStatus = 'pending' | 'contract' | 'completed' | 'cancelled';

export interface Deal {
  id: number;
  id_user: number;
  id_employee: number;
  id_apartment: number;
  id_chat_session?: number;
  base_price: number;
  percent_discount: number;
  total_price: number;
  status: DealStatus;
  created_at: string;
  updated_at: string;
  user_name?: string;
  employee_name?: string;
  apartment_number?: string;
}

export type OfferStatus = 'draft' | 'pending_approval' | 'approved' | 'rejected';

export interface Offer {
  id: number;
  deal_id: number;
  version: number;
  created_by: number;
  base_price: number;
  discount_percent: string;
  final_price: number;
  parking_unit_id?: number;
  parking_number?: string;
  parking_price?: number;
  storage_unit_id?: number;
  storage_number?: string;
  storage_price?: number;
  generated_text: string;
  status: OfferStatus;
  approval_required: boolean;
  approved_by?: number;
  approved_at?: string;
  rejected_by?: number;
  rejected_at?: string;
  rejection_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface Notification {
  id: number;
  user_id: number;
  deal_id?: number;
  type: 'construction_delay' | 'construction_risk' | 'deal_update' | 'general';
  title: string;
  message: string;
  is_read: boolean;
  created_at: string;
  read_at?: string;
}

export interface DiscountPolicy {
  id: number;
  building_id: number;
  role: Role;
  max_discount_percent: string;
  version: number;
  valid_from: string;
  valid_to?: string;
  created_by: number;
  created_at: string;
}

export interface ClientFacts {
  budget_min?: number;
  budget_max?: number;
  rooms?: number;
  floor_min?: number;
  floor_max?: number;
  parking_required?: boolean;
  renovation_required?: boolean;
  preferred_district?: string;
  purchase_timeline?: string;
  important_factors?: string[];
  objections?: string[];
  summary: string;
}

export interface DialogAnalysisResult {
  deal_id: number;
  client_id?: number;
  analysis?: ClientFacts;
  preferences_updated?: boolean;
  summary?: string;
  client_needs?: {
    rooms?: number;
    budget_max?: number;
    district?: string;
    preferred_finishing?: string;
  };
  objections?: string[];
  recommended_strategy?: string;
}

export interface ReplyAssistResult {
  deal_id: number;
  source?: string;
  suggested_reply: string;
  analysis?: {
    intent?: string;
    summary?: string;
    tone?: string;
    key_points?: string[];
  };
}

export interface AIChatResponse {
  message: string;
  agent?: string;
  intent?: string;
}

