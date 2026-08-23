export type InvoiceStatus = 'Open' | 'Closed';

export interface InvoiceItem {
  id: number;
  invoice_id: number;
  product_id: number;
  quantity: number;
  created_at: string;
}

export interface Invoice {
  id: number;
  sequential_number: number;
  status: InvoiceStatus;
  created_at: string;
  updated_at: string;
  items?: InvoiceItem[];
}

export interface CreateInvoiceItemPayload {
  product_id: number;
  quantity: number;
}

export interface CreateInvoicePayload {
  items: CreateInvoiceItemPayload[];
}
