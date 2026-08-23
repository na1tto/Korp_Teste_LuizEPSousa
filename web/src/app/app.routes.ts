import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', redirectTo: 'products', pathMatch: 'full' },
  {
    path: 'products',
    loadComponent: () =>
      import('./features/products/products.component').then(
        (m) => m.ProductsComponent
      ),
  },
  {
    path: 'invoices',
    loadComponent: () =>
      import('./features/invoices/invoices.component').then(
        (m) => m.InvoicesComponent
      ),
  },
  { path: '**', redirectTo: 'products' },
];
