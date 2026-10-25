export interface PaginationParams {
  page?: number;
  limit?: number;
  all?: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}
