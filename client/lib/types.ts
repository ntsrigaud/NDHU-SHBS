export type Condition = 'good' | 'moderate' | 'poor';
export type ListingStatus = 'active' | 'pending' | 'reserved' | 'sold' | 'delisted';

export interface ListingImage {
  id: string;
  listing_id: string;
  image_id: string;
  display_order: number;
  cdn_url: string;
}

export interface BookListing {
  id: string;
  seller_id: string;
  title: string;
  author: string;
  isbn: string | null;
  course_code: string | null;
  department: string | null;
  price: number;
  condition: Condition;
  status: ListingStatus;
  description: string | null;
  ai_confidence: number | null;
  condition_score: number | null;
  ai_processed: boolean;
  created_at: string;
  updated_at: string;
  images: ListingImage[];
  seller?: {
    id: string;
    name: string;
  };
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}

export interface ListingsFilter {
  search?: string;
  department?: string;
  condition?: Condition;
  price_min?: number;
  price_max?: number;
  page?: number;
  limit?: number;
}
