import { HttpErrorResponse } from '@angular/common/http';
import { ApiErrorResponse } from '../models/api-response.model';

const translatedBackendMessages: Record<string, string> = {
  'the server encountered a problem': 'O servidor encontrou um problema. Tente novamente.',
  'not found': 'Registro não encontrado.',
  'method not allowed': 'Operação não permitida.',
  'request timed out': 'A solicitação demorou demais. Tente novamente.',
  'a product with this code already exists': 'Já existe um produto com este código.',
  'product_id and quantity must be greater than zero':
    'O produto e a quantidade devem ser maiores que zero.',
  'insufficient stock for the product': 'Estoque insuficiente para o produto informado.',
  'the invoice must contain at least one item': 'A nota fiscal deve conter pelo menos um item.',
  'each item must have a valid product_id and quantity':
    'Todos os itens devem ter um produto e uma quantidade válidos.',
  'invoice items must have valid product_id and quantity values':
    'Todos os itens devem ter um produto e uma quantidade válidos.',
  'the invoice could not be created: insufficient available stock':
    'A nota fiscal não foi criada porque o estoque disponível é insuficiente.',
  'the invoice could not be issued: insufficient stock':
    'A nota fiscal não pôde ser emitida porque o estoque é insuficiente.',
  'the invoice contains a product that no longer exists':
    'A nota fiscal contém um produto que não existe mais.',
  "only invoices with the status 'Open' can be printed":
    'Somente notas fiscais abertas podem ser impressas.',
  'the invoice is no longer Open': 'A nota fiscal não está mais aberta.',
  'only Open invoices that have not been deducted can be deleted':
    'Somente notas fiscais abertas e ainda não debitadas podem ser excluídas.',
  'communication with the inventory service failed. The invoice remains Open and can be retried safely':
    'Não foi possível consultar o estoque. A nota permanece aberta e a operação pode ser tentada novamente.',
};

export function getUserErrorMessage(error: unknown, fallback: string): string {
  if (!(error instanceof HttpErrorResponse)) {
    return fallback;
  }

  const response = error.error as Partial<ApiErrorResponse> | null;
  const backendMessage = typeof response?.error === 'string' ? response.error : null;
  if (backendMessage && translatedBackendMessages[backendMessage]) {
    return translatedBackendMessages[backendMessage];
  }

  switch (error.status) {
    case 0:
      return 'Não foi possível se comunicar com o servidor.';
    case 400:
      return 'Os dados enviados são inválidos. Revise as informações e tente novamente.';
    case 404:
      return 'Registro não encontrado.';
    case 409:
      return 'A operação não pôde ser concluída devido a um conflito.';
    case 422:
      return 'Os dados informados não podem ser processados.';
    case 503:
      return 'O serviço está temporariamente indisponível. Tente novamente.';
    default:
      return fallback;
  }
}
