import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, catchError, map, tap, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Product, CreateProductPayload } from '../models/product.model';
import { ApiResponse } from '../models/api-response.model';
import { getUserErrorMessage } from '../utils/api-error';

@Injectable({
  providedIn: 'root',
})
export class ProductService {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = `${environment.stockApiUrl}/products`;

  // Reactive product state management
  private productsSubject = new BehaviorSubject<Product[]>([]);
  public readonly products$ = this.productsSubject.asObservable();

  // Retrieves the list of products and updates the BehaviorSubject.
  getProducts(): Observable<Product[]> {
    return this.http.get<ApiResponse<Product[]>>(this.apiUrl).pipe(
      map((response) => response.data),
      tap((products) => this.productsSubject.next(products)),
      catchError((error) => this.handleError(error, 'Não foi possível carregar os produtos.')),
    );
  }

  // Creates a new product in stock
  createProduct(payload: CreateProductPayload): Observable<Product> {
    return this.http.post<ApiResponse<Product>>(this.apiUrl, payload).pipe(
      map((response) => response.data),
      tap((product) => {
        const products = this.productsSubject.value.filter((current) => current.id !== product.id);
        this.productsSubject.next([...products, product]);
      }),
      catchError((error) => this.handleError(error, 'Não foi possível cadastrar o produto.')),
    );
  }

  private handleError(error: unknown, fallback: string): Observable<never> {
    console.error('ProductService error:', error);
    return throwError(() => new Error(getUserErrorMessage(error, fallback)));
  }
}
