import { Component, DestroyRef, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, ReactiveFormsModule, Validators } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { finalize, forkJoin, switchMap } from 'rxjs';
import { InvoiceService } from '../../core/services/invoice.service';
import { ProductService } from '../../core/services/product.service';
import { CreateInvoicePayload, Invoice, InvoiceStatus } from '../../core/models/invoice.model';
import { Product } from '../../core/models/product.model';

@Component({
  selector: 'app-invoices',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './invoices.component.html',
  styleUrls: ['./invoices.component.scss'],
})
export class InvoicesComponent implements OnInit {
  private fb = inject(FormBuilder);
  private invoiceService = inject(InvoiceService);
  private productService = inject(ProductService);
  private destroyRef = inject(DestroyRef);

  public invoiceForm!: FormGroup;
  public invoices$ = this.invoiceService.invoices$;
  public products: Product[] = [];

  public isLoading = false;
  public printingInvoiceId: number | null = null;
  public deletingInvoiceId: number | null = null;
  public invoicePendingDeletionId: number | null = null;
  public errorMessage: string | null = null;
  public successMessage: string | null = null;

  ngOnInit(): void {
    this.initForm();
    this.loadInitialData();
  }

  /**
   * Getter para facilitar a manipulação do FormArray de itens no template
   */
  get items(): FormArray {
    return this.invoiceForm.get('items') as FormArray;
  }

  private initForm(): void {
    this.invoiceForm = this.fb.group({
      items: this.fb.array([], [Validators.required, Validators.minLength(1)]),
    });

    // Adiciona o primeiro item por padrão
    this.addItem();
  }

  private loadInitialData(): void {
    this.isLoading = true;

    this.productService.products$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((products) => (this.products = products));

    forkJoin({
      products: this.productService.getProducts(),
      invoices: this.invoiceService.getInvoices(),
    })
      .pipe(
        finalize(() => (this.isLoading = false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        error: (error: Error) => (this.errorMessage = error.message),
      });
  }

  /**
   * Adiciona uma nova linha de produto dinamicamente via FormArray
   */
  public addItem(): void {
    const itemGroup = this.fb.group({
      product_id: ['', [Validators.required]],
      quantity: [1, [Validators.required, Validators.min(1)]],
    });
    this.items.push(itemGroup);
  }

  /**
   * Remove uma linha de item do FormArray
   */
  public removeItem(index: number): void {
    if (this.items.length > 1) {
      this.items.removeAt(index);
    }
  }

  /**
   * Cria uma nova nota fiscal com status inicial 'Aberta'
   */
  public onSubmit(): void {
    if (this.invoiceForm.invalid) {
      this.invoiceForm.markAllAsTouched();
      return;
    }

    this.isLoading = true;
    this.errorMessage = null;
    this.successMessage = null;

    const payload: CreateInvoicePayload = {
      items: this.items.getRawValue().map((item) => ({
        product_id: Number(item.product_id),
        quantity: Number(item.quantity),
      })),
    };

    this.invoiceService
      .createInvoice(payload)
      .pipe(
        finalize(() => (this.isLoading = false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: () => {
          this.successMessage = 'Nota fiscal cadastrada com sucesso. Status: aberta.';
          this.items.clear();
          this.addItem();
        },
        error: (error: Error) => (this.errorMessage = error.message),
      });
  }

  /**
   * Executa a impressão da Nota Fiscal com tratamento de erro e loading individual
   */
  public printInvoice(invoice: Invoice): void {
    if (
      invoice.status !== 'Open' ||
      this.printingInvoiceId !== null ||
      this.deletingInvoiceId !== null
    ) {
      return;
    }

    this.printingInvoiceId = invoice.id;
    this.errorMessage = null;
    this.successMessage = null;

    this.invoiceService
      .printInvoice(invoice.id)
      .pipe(
        switchMap(() => this.productService.getProducts()),
        finalize(() => (this.printingInvoiceId = null)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: () => {
          this.successMessage = `Nota fiscal nº ${invoice.sequential_number} impressa e fechada com sucesso.`;
        },
        error: (error: Error) => (this.errorMessage = error.message),
      });
  }

  public requestInvoiceDeletion(invoice: Invoice): void {
    if (
      invoice.status !== 'Open' ||
      this.printingInvoiceId !== null ||
      this.deletingInvoiceId !== null
    ) {
      return;
    }

    this.errorMessage = null;
    this.successMessage = null;
    this.invoicePendingDeletionId = invoice.id;
  }

  public cancelInvoiceDeletion(): void {
    this.invoicePendingDeletionId = null;
  }

  public confirmInvoiceDeletion(invoice: Invoice): void {
    if (this.invoicePendingDeletionId !== invoice.id || invoice.status !== 'Open') {
      return;
    }

    this.invoicePendingDeletionId = null;
    this.deletingInvoiceId = invoice.id;
    this.errorMessage = null;
    this.successMessage = null;

    this.invoiceService
      .deleteInvoice(invoice.id)
      .pipe(
        finalize(() => (this.deletingInvoiceId = null)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: () => {
          this.successMessage = `Nota fiscal nº ${invoice.sequential_number} excluída com sucesso.`;
        },
        error: (error: Error) => (this.errorMessage = error.message),
      });
  }

  public getStatusLabel(status: InvoiceStatus): string {
    return status === 'Open' ? 'Aberta' : 'Fechada';
  }

  public getProductDescription(productId: number): string {
    const prod = this.products.find((p) => p.id === productId);
    return prod ? `${prod.code} - ${prod.description}` : `Produto #${productId}`;
  }
}
