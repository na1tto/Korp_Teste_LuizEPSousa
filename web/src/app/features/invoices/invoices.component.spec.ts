import { TestBed } from '@angular/core/testing';
import { BehaviorSubject, of } from 'rxjs';
import { Invoice } from '../../core/models/invoice.model';
import { InvoiceService } from '../../core/services/invoice.service';
import { ProductService } from '../../core/services/product.service';
import { InvoicesComponent } from './invoices.component';

describe('InvoicesComponent', () => {
  const invoice: Invoice = {
    id: 7,
    sequential_number: 100,
    status: 'Open',
    created_at: '2026-08-22T12:00:00Z',
    updated_at: '2026-08-22T12:00:00Z',
  };

  const invoicesSubject = new BehaviorSubject<Invoice[]>([invoice]);
  const productsSubject = new BehaviorSubject([]);
  const invoiceService = {
    invoices$: invoicesSubject.asObservable(),
    getInvoices: vi.fn(() => of([invoice])),
    createInvoice: vi.fn(),
    printInvoice: vi.fn(),
    deleteInvoice: vi.fn(() => {
      invoicesSubject.next([]);
      return of(undefined);
    }),
  };
  const productService = {
    products$: productsSubject.asObservable(),
    getProducts: vi.fn(() => of([])),
  };

  beforeEach(async () => {
    invoicesSubject.next([invoice]);
    vi.clearAllMocks();
    await TestBed.configureTestingModule({
      imports: [InvoicesComponent],
      providers: [
        { provide: InvoiceService, useValue: invoiceService },
        { provide: ProductService, useValue: productService },
      ],
    }).compileComponents();
  });

  it('deve confirmar e excluir uma nota aberta', async () => {
    const fixture = TestBed.createComponent(InvoicesComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    const deleteButton = fixture.nativeElement.querySelector('.btn-delete') as HTMLButtonElement;
    deleteButton.click();
    fixture.detectChanges();

    const confirmation = fixture.nativeElement.querySelector('.delete-confirmation');
    expect(confirmation?.textContent).toContain('Excluir esta nota?');

    const confirmButton = fixture.nativeElement.querySelector(
      '.btn-confirm-delete',
    ) as HTMLButtonElement;
    confirmButton.click();
    fixture.detectChanges();

    expect(invoiceService.deleteInvoice).toHaveBeenCalledWith(invoice.id);
    expect(fixture.componentInstance.successMessage).toContain('excluída com sucesso');
  });
});
