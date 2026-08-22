import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, BehaviorSubject, catchError, tap, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Product, CreateProductPayload } from '../models/product.model';

@Injectable({
  providedIn: 'root',
})
export class ProductService {
  private http = inject(HttpClient);
  private apiUrl = `${environment.stockApiUrl}/products`;

  // Reactive product state management
  private productsSubject = new BehaviorSubject<Product[]>([]);
  public products$ = this.productsSubject.asObservable();

  // Retrieves the list of products and updates the BehaviorSubject.
  getProducts(): Observable<Product[]> {
    return this.http.get<Product[]>(this.apiUrl).pipe(
      tap((products) => this.productsSubject.next(products)),
      catchError(this.handleError)
    );
  }

  // Creates a new product in stock
  createProduct(payload: CreateProductPayload): Observable<Product> {
    return this.http.post<Product>(this.apiUrl, payload).pipe(
      tap(() => this.getProducts().subscribe()), // Reloads the list reactively
      catchError(this.handleError)
    );
  }

  private handleError(error: any) {
    console.error('ProductService error:', error);
    return throwError(() => new Error(error.error?.error || 'Communication with the inventory department error'));
  }
}
