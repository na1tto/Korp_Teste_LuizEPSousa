import { TestBed } from '@angular/core/testing';
import { BehaviorSubject, Observable, Subject, of, tap } from 'rxjs';
import { Product } from '../../core/models/product.model';
import { ProductService } from '../../core/services/product.service';
import { ProductsComponent } from './products.component';

describe('ProductsComponent', () => {
  const product: Product = {
    id: 3,
    code: 'PROD-3',
    description: 'Produto de teste',
    balance: 1,
    created_at: '2026-08-22T12:00:00Z',
    updated_at: '2026-08-22T12:00:00Z',
  };

  let productsSubject: BehaviorSubject<Product[]>;
  let createResponse: Subject<Product>;
  let productService: {
    products$: Observable<Product[]>;
    getProducts: ReturnType<typeof vi.fn>;
    createProduct: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    productsSubject = new BehaviorSubject<Product[]>([]);
    createResponse = new Subject<Product>();
    productService = {
      products$: productsSubject.asObservable(),
      getProducts: vi.fn(() => of([])),
      createProduct: vi.fn(() =>
        createResponse.pipe(tap((createdProduct) => productsSubject.next([createdProduct]))),
      ),
    };

    await TestBed.configureTestingModule({
      imports: [ProductsComponent],
      providers: [{ provide: ProductService, useValue: productService }],
    }).compileComponents();
  });

  it('renderiza o produto e o feedback sem detecção manual após a resposta', async () => {
    const fixture = TestBed.createComponent(ProductsComponent);
    fixture.autoDetectChanges();
    await fixture.whenStable();

    fixture.componentInstance.productForm.setValue({
      code: product.code,
      description: product.description,
      balance: product.balance,
    });
    fixture.detectChanges();

    const submitButton = fixture.nativeElement.querySelector('.btn-submit') as HTMLButtonElement;
    submitButton.click();
    await fixture.whenStable();
    expect(submitButton.textContent).toContain('Salvando...');

    createResponse.next(product);
    createResponse.complete();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.alert-success')?.textContent).toContain(
      'Produto cadastrado com sucesso.',
    );
    expect(fixture.nativeElement.querySelector('.products-table')?.textContent).toContain(
      product.description,
    );
    expect(fixture.nativeElement.textContent).not.toContain('Salvando...');
  });
});
