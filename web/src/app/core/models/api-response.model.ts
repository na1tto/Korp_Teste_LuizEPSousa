export interface ApiResponse<T> {
  data: T;
}

export interface ApiErrorResponse {
  error: string;
}
