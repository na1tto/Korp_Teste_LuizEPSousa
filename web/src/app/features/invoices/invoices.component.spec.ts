import { TestBed } from '@angular/core/testing';
import { BehaviorSubject, Observable, Subject, of, tap } from 'rxjs';
import { Invoice } from '../../core/models/invoice.model';
import { Product } from '../../core/models/product.model';
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
  const product: Product = {
    id: 3,
    code: 'PROD-3',
    description: 'Produto de teste',
    balance: 1,
    created_at: '2026-08-22T12:00:00Z',
    updated_at: '2026-08-22T12:00:00Z',
  };

  let invoicesSubject: BehaviorSubject<Invoice[]>;
  let productsSubject: BehaviorSubject<Product[]>;
  let createResponse: Subject<Invoice>;
  let printResponse: Subject<Invoice>;
  let deleteResponse: Subject<void>;
  let productRefreshResponse: Subject<Product[]>;
  let invoiceService: {
    invoices$: Observable<Invoice[]>;
    getInvoices: ReturnType<typeof vi.fn>;
    createInvoice: ReturnType<typeof vi.fn>;
    printInvoice: ReturnType<typeof vi.fn>;
    deleteInvoice: ReturnType<typeof vi.fn>;
  };
  let productService: {
    products$: Observable<Product[]>;
    getProducts: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    invoicesSubject = new BehaviorSubject<Invoice[]>([invoice]);
    productsSubject = new BehaviorSubject<Product[]>([product]);
    createResponse = new Subject<Invoice>();
    printResponse = new Subject<Invoice>();
    deleteResponse = new Subject<void>();
    productRefreshResponse = new Subject<Product[]>();

    invoiceService = {
      invoices$: invoicesSubject.asObservable(),
      getInvoices: vi.fn(() => of([invoice])),
      createInvoice: vi.fn(() =>
        createResponse.pipe(tap((createdInvoice) => invoicesSubject.next([createdInvoice]))),
      ),
      printInvoice: vi.fn(() =>
        printResponse.pipe(tap((printedInvoice) => invoicesSubject.next([printedInvoice]))),
      ),
      deleteInvoice: vi.fn(() => deleteResponse.pipe(tap(() => invoicesSubject.next([])))),
    };
    productService = {
      products$: productsSubject.asObservable(),
      getProducts: vi
        .fn()
        .mockReturnValueOnce(of([product]))
        .mockReturnValueOnce(
          productRefreshResponse.pipe(tap((products) => productsSubject.next(products))),
        ),
    };

    await TestBed.configureTestingModule({
      imports: [InvoicesComponent],
      providers: [
        { provide: InvoiceService, useValue: invoiceService },
        { provide: ProductService, useValue: productService },
      ],
    }).compileComponents();
  });

  it('renderiza a exclusão e o feedback sem detecção manual após a resposta', async () => {
    const fixture = TestBed.createComponent(InvoicesComponent);
    fixture.autoDetectChanges();
    await fixture.whenStable();

    const deleteButton = fixture.nativeElement.querySelector('.btn-delete') as HTMLButtonElement;
    deleteButton.click();
    await fixture.whenStable();

    const confirmButton = fixture.nativeElement.querySelector(
      '.btn-confirm-delete',
    ) as HTMLButtonElement;
    confirmButton.click();
    await fixture.whenStable();
    expect(fixture.nativeElement.textContent).toContain('Excluindo...');

    deleteResponse.next();
    deleteResponse.complete();
    await fixture.whenStable();

    expect(invoiceService.deleteInvoice).toHaveBeenCalledWith(invoice.id);
    expect(fixture.nativeElement.querySelector('.alert-success')?.textContent).toContain(
      'excluída com sucesso',
    );
    expect(fixture.nativeElement.querySelector('.empty-state')?.textContent).toContain(
      'Nenhuma nota fiscal registrada',
    );
  });

  it('renderiza o feedback de criação sem interação adicional', async () => {
    const fixture = TestBed.createComponent(InvoicesComponent);
    fixture.autoDetectChanges();
    await fixture.whenStable();

    fixture.componentInstance.items.at(0).setValue({ product_id: product.id, quantity: 1 });
    await fixture.whenStable();

    const submitButton = fixture.nativeElement.querySelector('.btn-submit') as HTMLButtonElement;
    submitButton.click();
    await fixture.whenStable();
    expect(submitButton.textContent).toContain('Processando...');

    createResponse.next(invoice);
    createResponse.complete();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.alert-success')?.textContent).toContain(
      'Nota fiscal cadastrada com sucesso.',
    );
    expect(fixture.nativeElement.textContent).not.toContain('Processando...');
  });

  it('renderiza o feedback de impressão junto com o status e o saldo atualizados', async () => {
    const fixture = TestBed.createComponent(InvoicesComponent);
    fixture.autoDetectChanges();
    await fixture.whenStable();

    const printButton = fixture.nativeElement.querySelector('.btn-print') as HTMLButtonElement;
    printButton.click();
    await fixture.whenStable();
    expect(printButton.textContent).toContain('Processando...');

    printResponse.next({ ...invoice, status: 'Closed' });
    printResponse.complete();
    productRefreshResponse.next([{ ...product, balance: 0 }]);
    productRefreshResponse.complete();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.alert-success')?.textContent).toContain(
      'impressa e fechada com sucesso',
    );
    expect(fixture.nativeElement.querySelector('.status-badge')?.textContent).toContain('Fechada');
    expect(fixture.nativeElement.querySelector('select')?.textContent).toContain('Saldo: 0');
    expect(fixture.nativeElement.textContent).not.toContain('Processando...');
  });

  it('renderiza erros assíncronos sem interação adicional', async () => {
    const fixture = TestBed.createComponent(InvoicesComponent);
    fixture.autoDetectChanges();
    await fixture.whenStable();

    const printButton = fixture.nativeElement.querySelector('.btn-print') as HTMLButtonElement;
    printButton.click();
    printResponse.error(new Error('Não foi possível imprimir a nota fiscal.'));
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.alert-danger')?.textContent).toContain(
      'Não foi possível imprimir a nota fiscal.',
    );
    expect(fixture.componentInstance.printingInvoiceId()).toBeNull();
  });
});
