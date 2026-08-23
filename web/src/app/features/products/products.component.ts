import { Component, OnInit, inject, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { finalize } from 'rxjs';
import { ProductService } from '../../core/services/product.service';
import { CreateProductPayload } from '../../core/models/product.model';

@Component({
  selector: 'app-products',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './products.component.html',
  styleUrls: ['./products.component.scss'],
})
export class ProductsComponent implements OnInit {
  private fb = inject(FormBuilder);
  private productService = inject(ProductService);
  private destroyRef = inject(DestroyRef); // Modern Unsubscribe Management

  public productForm!: FormGroup;
  public products$ = this.productService.products$;
  public isLoading = false;
  public errorMessage: string | null = null;
  public successMessage: string | null = null;

  ngOnInit(): void {
    this.initForm();
    this.loadProducts();
  }

  /**
   * Initializes the Reactive Form with synchronous controls and validations
   */
  private initForm(): void {
    this.productForm = this.fb.group({
      code: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(50)]],
      description: ['', [Validators.required, Validators.maxLength(255)]],
      balance: [0, [Validators.required, Validators.min(0)]],
    });
  }

  /**
   * Startup cycle: searching for products using takeUntilDestroyed to prevent leaks
   */
  private loadProducts(): void {
    this.isLoading = true;
    this.productService
      .getProducts()
      .pipe(
        finalize(() => (this.isLoading = false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        error: (error: Error) => (this.errorMessage = error.message),
      });
  }

  /**
   * Form submission
   */
  public onSubmit(): void {
    if (this.productForm.invalid) {
      this.productForm.markAllAsTouched();
      return;
    }

    this.isLoading = true;
    this.errorMessage = null;
    this.successMessage = null;

    const formValue = this.productForm.getRawValue();
    const payload: CreateProductPayload = {
      code: String(formValue.code).trim(),
      description: String(formValue.description).trim(),
      balance: Number(formValue.balance),
    };

    this.productService
      .createProduct(payload)
      .pipe(
        finalize(() => (this.isLoading = false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: () => {
          this.successMessage = 'Produto cadastrado com sucesso.';
          this.productForm.reset({ balance: 0 });
        },
        error: (error: Error) => (this.errorMessage = error.message),
      });
  }

  // Quick access helpers for form controls in the template
  get f() {
    return this.productForm.controls;
  }
}
