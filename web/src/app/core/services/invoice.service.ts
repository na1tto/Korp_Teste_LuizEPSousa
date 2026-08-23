import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, catchError, map, tap, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Invoice, CreateInvoicePayload } from '../models/invoice.model';
import { ApiResponse } from '../models/api-response.model';
import { getUserErrorMessage } from '../utils/api-error';

@Injectable({
  providedIn: 'root',
})
export class InvoiceService {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = `${environment.invoicingApiUrl}/invoices`;

  private invoicesSubject = new BehaviorSubject<Invoice[]>([]);
  public readonly invoices$ = this.invoicesSubject.asObservable();

  // Lists all invoices
  getInvoices(): Observable<Invoice[]> {
    return this.http.get<ApiResponse<Invoice[]>>(this.apiUrl).pipe(
      map((response) => response.data),
      tap((invoices) => this.invoicesSubject.next(invoices)),
      catchError((error) => this.handleError(error, 'Não foi possível carregar as notas fiscais.')),
    );
  }
  // Search for invoice details by ID
  getInvoiceById(id: number): Observable<Invoice> {
    return this.http.get<ApiResponse<Invoice>>(`${this.apiUrl}/${id}`).pipe(
      map((response) => response.data),
      catchError((error) => this.handleError(error, 'Não foi possível carregar a nota fiscal.')),
    );
  }

  // Creates a new invoice with initial status 'Open'
  createInvoice(payload: CreateInvoicePayload): Observable<Invoice> {
    return this.http.post<ApiResponse<Invoice>>(this.apiUrl, payload).pipe(
      map((response) => response.data),
      tap((invoice) => {
        const invoices = this.invoicesSubject.value.filter((current) => current.id !== invoice.id);
        this.invoicesSubject.next([invoice, ...invoices]);
      }),
      catchError((error) => this.handleError(error, 'Não foi possível cadastrar a nota fiscal.')),
    );
  }

  // Executes the printing flow: validation -> stock reduction -> invoice closing
  printInvoice(id: number): Observable<Invoice> {
    return this.http.post<ApiResponse<Invoice>>(`${this.apiUrl}/${id}/print`, {}).pipe(
      map((response) => response.data),
      tap((invoice) => {
        const invoices = this.invoicesSubject.value.map((current) =>
          current.id === invoice.id ? invoice : current,
        );
        this.invoicesSubject.next(invoices);
      }),
      catchError((error) => this.handleError(error, 'Não foi possível imprimir a nota fiscal.')),
    );
  }

  deleteInvoice(id: number): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/${id}`).pipe(
      tap(() => {
        const invoices = this.invoicesSubject.value.filter((invoice) => invoice.id !== id);
        this.invoicesSubject.next(invoices);
      }),
      catchError((error) => this.handleError(error, 'Não foi possível excluir a nota fiscal.')),
    );
  }

  private handleError(error: unknown, fallback: string): Observable<never> {
    console.error('InvoiceService error:', error);
    return throwError(() => new Error(getUserErrorMessage(error, fallback)));
  }
}
