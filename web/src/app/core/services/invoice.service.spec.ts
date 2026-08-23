import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Invoice } from '../models/invoice.model';
import { environment } from '../../../environments/environment';
import { InvoiceService } from './invoice.service';

describe('InvoiceService', () => {
  let service: InvoiceService;
  let http: HttpTestingController;

  const invoice: Invoice = {
    id: 7,
    sequential_number: 100,
    status: 'Open',
    created_at: '2026-08-22T12:00:00Z',
    updated_at: '2026-08-22T12:00:00Z',
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    service = TestBed.inject(InvoiceService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('deve excluir a nota e atualizar o estado reativo', () => {
    let invoices: Invoice[] = [];
    service.invoices$.subscribe((value) => (invoices = value));

    service.getInvoices().subscribe();
    http.expectOne(`${environment.invoicingApiUrl}/invoices`).flush({ data: [invoice] });

    service.deleteInvoice(invoice.id).subscribe();
    const request = http.expectOne(`${environment.invoicingApiUrl}/invoices/${invoice.id}`);
    expect(request.request.method).toBe('DELETE');
    request.flush(null, { status: 204, statusText: 'No Content' });

    expect(invoices).toEqual([]);
  });

  it('deve traduzir conflito de exclusão para português', () => {
    let message = '';
    service.deleteInvoice(invoice.id).subscribe({
      error: (error: Error) => (message = error.message),
    });

    http
      .expectOne(`${environment.invoicingApiUrl}/invoices/${invoice.id}`)
      .flush(
        { error: 'only Open invoices that have not been deducted can be deleted' },
        { status: 409, statusText: 'Conflict' },
      );

    expect(message).toBe(
      'Somente notas fiscais abertas e ainda não debitadas podem ser excluídas.',
    );
  });
});
