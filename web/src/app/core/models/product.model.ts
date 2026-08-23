export interface Product {
  id: number;
  code: string;
  description: string;
  balance: number;
  created_at: string;
  updated_at: string;
}

export interface CreateProductPayload {
  code: string;
  description: string;
  balance: number;
}
