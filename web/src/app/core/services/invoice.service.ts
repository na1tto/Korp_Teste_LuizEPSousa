import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, BehaviorSubject, catchError, tap, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Invoice, CreateInvoicePayload } from '../models/invoice.model';

@Injectable({
  providedIn: 'root',
})
export class InvoiceService {
  private http = inject(HttpClient);
  private apiUrl = `${environment.invoicingApiUrl}/invoices`;

  private invoicesSubject = new BehaviorSubject<Invoice[]>([]);
  public invoices$ = this.invoicesSubject.asObservable();

  // Lists all invoices
  getInvoices(): Observable<Invoice[]> {
    return this.http.get<Invoice[]>(this.apiUrl).pipe(
      tap((invoices) => this.invoicesSubject.next(invoices)),
      catchError(this.handleError)
    );
  }
  // Search for invoice details by ID
  getInvoiceById(id: number): Observable<Invoice> {
    return this.http.get<Invoice>(`${this.apiUrl}/${id}`).pipe(
      catchError(this.handleError)
    );
  }


  // Creates a new invoice with initial status 'Open'
  createInvoice(payload: CreateInvoicePayload): Observable<Invoice> {
    return this.http.post<Invoice>(this.apiUrl, payload).pipe(
      tap(() => this.getInvoices().subscribe()),
      catchError(this.handleError)
    );
  }

  // Executes the printing flow: validation -> stock reduction -> invoice closing
  printInvoice(id: number): Observable<Invoice> {
    return this.http.post<Invoice>(`${this.apiUrl}/${id}/print`, {}).pipe(
      tap(() => this.getInvoices().subscribe()),
      catchError(this.handleError)
    );
  }

  private handleError(error: any) {
    console.error('InvoiceService error:', error);
    const message = error.error?.error || 'Error processing billing request';
    return throwError(() => new Error(message));
  }
}
