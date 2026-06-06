import {
  Api as GeneratedApi,
  type HttpResponse,
  type ModelCartItemResponse,
  type ModelConversationResponse,
  type ModelListingWithImages,
  type ModelMessageResponse,
  type ModelNotificationResponse,
  type ModelOrderResponse,
  type ModelSwaggerErrorResponse,
  type ModelSwaggerLoginRequest,
  type ModelSwaggerLoginResponse,
  type ModelSwaggerMessageResponse,
  type ModelSwaggerRegisterRequest,
  type ModelSwaggerUpdateListingRequest,
  type ModelUser,
} from '@/generated/api';
import type { BookListing, ListingsFilter } from '@/lib/types';
import type { CartItem } from '@/slices/cartSlice';
import type { User } from '@/slices/authSlice';

type SecurityData = {
  token?: string;
};

export class ApiError extends Error {
  status: number;
  payload: unknown;

  constructor(message: string, status: number, payload: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.payload = payload;
  }
}

function getServerBaseUrl(): string {
  const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
  return baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
}

function getApiBaseUrl(): string {
  const baseUrl = getServerBaseUrl();
  return baseUrl.endsWith('/api/v1') ? baseUrl : `${baseUrl}/api/v1`;
}

export const SERVER_BASE_URL = getServerBaseUrl();

function createApi(token?: string) {
  const api = new GeneratedApi<SecurityData>({
    baseUrl: getApiBaseUrl(),
    baseApiParams: {
      credentials: 'include',
    },
    securityWorker: (securityData) => {
      if (!securityData?.token) {
        return undefined;
      }

      return {
        headers: {
          Authorization: `Bearer ${securityData.token}`,
        },
      };
    },
  });

  api.setSecurityData(token ? { token } : null);

  return api;
}

export type ApiClient = ReturnType<typeof createApi>;
export type ApiRequest<T> = Promise<HttpResponse<T, ModelSwaggerErrorResponse>>;
export type LoginInput = ModelSwaggerLoginRequest;
export type RegisterInput = ModelSwaggerRegisterRequest;
export type UpdateListingInput = ModelSwaggerUpdateListingRequest;

export interface AuthLoginResult extends Omit<ModelSwaggerLoginResponse, 'user'> {
  user?: User;
}

export interface Message {
  id: string;
  listing_id: string;
  sender_id: string;
  receiver_id: string;
  body: string;
  is_read: boolean;
  created_at: string;
  sender?: { id: string; name: string };
}

export interface Conversation {
  listing_id: string;
  listing_title: string;
  other_user: { id: string; name: string };
  last_message: string;
  last_message_at: string;
  unread_count: number;
}

export interface Notification {
  id: string;
  type: 'new_message' | 'order_confirmed' | 'listing_sold' | string;
  payload: Record<string, unknown>;
  is_read: boolean;
  created_at: string;
}

export interface Order {
  id: string;
  status: string;
  total: number;
  created_at: string;
  items: {
    listing_id: string;
    title: string;
    price: number;
    image_url: string;
  }[];
}

export interface UploadedImage {
  id: string;
  preview: string;
  cdn_url: string;
}

const api = createApi();

function toAuthUser(user: ModelUser): User {
  return {
    id: user.id ?? '',
    email: user.email ?? '',
    name: user.name ?? '',
    email_verified: Boolean(user.email_verified),
    is_admin: Boolean(user.is_admin),
  };
}

function toBookListing(listing: ModelListingWithImages): BookListing {
  return {
    id: listing.id ?? '',
    seller_id: listing.seller_id ?? '',
    title: listing.title ?? '',
    author: listing.author ?? '',
    isbn: listing.isbn ?? null,
    course_code: listing.course_code ?? null,
    department: listing.department ?? null,
    price: listing.price ?? 0,
    condition: (listing.condition as BookListing['condition']) ?? 'good',
    status: (listing.status as BookListing['status']) ?? 'active',
    description: listing.description ?? null,
    ai_confidence: listing.ai_confidence ?? null,
    created_at: listing.created_at ?? '',
    updated_at: listing.updated_at ?? '',
    images: (listing.image_urls ?? []).map((cdnUrl, index) => ({
      id: `${listing.id ?? 'listing'}-${index}`,
      listing_id: listing.id ?? '',
      image_id: `${index}`,
      display_order: index,
      cdn_url: cdnUrl,
    })),
    seller: listing.seller_id
      ? {
          id: listing.seller_id,
          name: listing.seller_name ?? 'Seller',
        }
      : undefined,
  };
}

function toCartItem(item: ModelCartItemResponse): CartItem {
  const listing = item.listing;

  return {
    listingId: listing?.id ?? '',
    title: listing?.title ?? '',
    price: listing?.price ?? 0,
    imageUrl: listing?.image_urls?.[0] ?? '',
    sellerName: listing?.seller_name ?? 'Seller',
  };
}

function toMessage(message: ModelMessageResponse): Message {
  return {
    id: message.id ?? '',
    listing_id: message.listing_id ?? '',
    sender_id: message.sender_id ?? '',
    receiver_id: message.receiver_id ?? '',
    body: message.body ?? '',
    is_read: Boolean(message.is_read),
    created_at: message.created_at ?? '',
    sender: message.sender_id
      ? {
          id: message.sender_id,
          name: message.sender_name ?? 'User',
        }
      : undefined,
  };
}

function toConversation(conversation: ModelConversationResponse): Conversation {
  return {
    listing_id: conversation.listing_id ?? '',
    listing_title: conversation.listing_title ?? '',
    other_user: {
      id: conversation.other_user_id ?? '',
      name: conversation.other_user_name ?? 'User',
    },
    last_message: conversation.last_message ?? '',
    last_message_at: conversation.last_message_at ?? '',
    unread_count: conversation.unread_count ?? 0,
  };
}

function toNotificationPayload(
  payload: ModelNotificationResponse['payload']
): Record<string, unknown> {
  if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
    return payload as unknown as Record<string, unknown>;
  }

  return {};
}

function toNotification(notification: ModelNotificationResponse): Notification {
  return {
    id: notification.id ?? '',
    type: notification.type ?? 'unknown',
    payload: toNotificationPayload(notification.payload),
    is_read: Boolean(notification.is_read),
    created_at: notification.created_at ?? '',
  };
}

function toOrder(order: ModelOrderResponse): Order {
  return {
    id: order.id ?? '',
    status: order.status ?? 'pending',
    total: order.total_amount ?? 0,
    created_at: order.created_at ?? '',
    items: (order.items ?? []).map((item) => ({
      listing_id: item.listing?.id ?? item.id ?? '',
      title: item.listing?.title ?? 'Listing',
      price: item.price_at_purchase ?? 0,
      image_url: item.listing?.image_urls?.[0] ?? '',
    })),
  };
}

function getCountValue(response: Record<string, number>): number {
  if (typeof response.count === 'number') {
    return response.count;
  }

  const fallback = Object.values(response)[0];
  return typeof fallback === 'number' ? fallback : 0;
}

function toApiError(error: unknown): ApiError {
  const httpError = error as Partial<HttpResponse<unknown, ModelSwaggerErrorResponse>>;
  const payload = httpError?.error;
  const message = payload?.error ?? (error instanceof Error ? error.message : 'Request failed');

  return new ApiError(message, httpError?.status ?? 500, payload);
}

export function getAuthenticatedApi(token?: string | null) {
  return createApi(token ?? undefined);
}

export async function executeApiResponse<T>(
  operation: (apiClient: ApiClient) => ApiRequest<T>
): Promise<HttpResponse<T, ModelSwaggerErrorResponse>> {
  try {
    return await operation(api);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function executeApiRequest<T>(
  operation: (apiClient: ApiClient) => ApiRequest<T>
): Promise<T> {
  const response = await executeApiResponse(operation);
  return response.data;
}

export async function executeAuthenticatedApiRequest<T>(
  accessToken: string,
  operation: (apiClient: ApiClient) => ApiRequest<T>
): Promise<T> {
  try {
    const response = await operation(getAuthenticatedApi(accessToken));
    return response.data;
  } catch (error) {
    throw toApiError(error);
  }
}

export const authApi = {
  async login(data: LoginInput): Promise<AuthLoginResult> {
    const response = await executeApiRequest((apiClient) => apiClient.auth.loginUser(data));

    return {
      ...response,
      user: response.user ? toAuthUser(response.user) : undefined,
    };
  },
  register(data: RegisterInput): Promise<ModelSwaggerMessageResponse> {
    return executeApiRequest((apiClient) => apiClient.auth.registerUser(data));
  },
  logout(): Promise<ModelSwaggerMessageResponse> {
    return executeApiRequest((apiClient) => apiClient.auth.logoutUser());
  },
  verifyEmail(token: string): Promise<ModelSwaggerMessageResponse> {
    return executeApiRequest((apiClient) => apiClient.auth.verifyEmail({ token }));
  },

export const imagesApi = {
  async uploadImage(file: File): Promise<UploadedImage> {
    const response = await executeApiRequest((apiClient) => apiClient.images.uploadImage({ file }));
    return {
      id: response.image?.id ?? '',
      preview: response.image?.cdn_url ?? '',
      cdn_url: response.image?.cdn_url ?? '',
    };
  },
};

export const listingsApi = {
  async getListings(filters: ListingsFilter): Promise<{ listings: BookListing[]; total: number }> {
    const response = await executeApiResponse((apiClient) =>
      apiClient.listings.getListings({
        status: 'active',
        department: filters.department,
        condition: filters.condition,
        price_min: filters.price_min,
        price_max: filters.price_max,
        page: filters.page ?? 1,
        limit: filters.limit ?? 20,
      })
    );

    return {
      listings: (response.data ?? []).map(toBookListing),
      total: Number(response.headers.get('X-Total-Count') ?? 0),
    };
  },
  async getListing(id: string): Promise<BookListing> {
    const listing = await executeApiRequest((apiClient) => apiClient.listings.getListing(id));
    return toBookListing(listing);
  },
  async getMyListings(): Promise<BookListing[]> {
    const listings = await executeApiRequest((apiClient) => apiClient.listings.getMyListings());
    return listings.map(toBookListing);
  },
  async createListing(data: Parameters<ApiClient['listings']['createListing']>[0]): Promise<BookListing> {
    const listing = await executeApiRequest((apiClient) => apiClient.listings.createListing(data));
    return toBookListing(listing);
  },
  async updateListing(
    id: string,
    data: UpdateListingInput
  ): Promise<BookListing> {
    const listing = await executeApiRequest((apiClient) => apiClient.listings.updateListing(id, data));
    return toBookListing(listing);
  },
  deleteListing(id: string): Promise<void> {
    return executeApiRequest((apiClient) => apiClient.listings.deleteListing(id));
  },
};

export const cartApi = {
  async getCart(): Promise<CartItem[]> {
    const items = await executeApiRequest((apiClient) => apiClient.cart.getCart());
    return items.map(toCartItem);
  },
  addToCart(listingId: string): Promise<ModelSwaggerMessageResponse> {
    return executeApiRequest((apiClient) => apiClient.cart.addToCart({ listing_id: listingId }));
  },
  removeFromCart(listingId: string): Promise<ModelSwaggerMessageResponse> {
    return executeApiRequest((apiClient) => apiClient.cart.removeFromCart(listingId));
  },
};

export const messagesApi = {
  async getMessages(listingId: string): Promise<Message[]> {
    const messages = await executeApiRequest((apiClient) => apiClient.listings.getMessages(listingId));
    return messages.map(toMessage);
  },
  async sendMessage(listingId: string, body: string): Promise<Message> {
    const message = await executeApiRequest((apiClient) =>
      apiClient.listings.sendMessage(listingId, { body })
    );
    return toMessage(message);
  },
  markRead(messageId: string): Promise<void> {
    return executeApiRequest((apiClient) => apiClient.messages.markMessageAsRead(messageId));
  },
  async getUnreadCount(): Promise<number> {
    const response = await executeApiRequest((apiClient) => apiClient.messages.getUnreadMessageCount());
    return getCountValue(response);
  },
  async getConversations(): Promise<Conversation[]> {
    const conversations = await executeApiRequest((apiClient) => apiClient.messages.getConversations());
    return conversations.map(toConversation);
  },
};

export const notificationsApi = {
  async getNotifications(): Promise<Notification[]> {
    const notifications = await executeApiRequest((apiClient) => apiClient.notifications.getNotifications());
    return notifications.map(toNotification);
  },
  async getUnreadCount(): Promise<number> {
    const response = await executeApiRequest((apiClient) =>
      apiClient.notifications.getUnreadNotificationCount()
    );
    return getCountValue(response);
  },
  markRead(id: string): Promise<void> {
    return executeApiRequest((apiClient) => apiClient.notifications.markNotificationAsRead(id));
  },
  markAllRead(): Promise<void> {
    return executeApiRequest((apiClient) => apiClient.notifications.markAllNotificationsAsRead());
  },
};

export const ordersApi = {
  async getOrders(): Promise<Order[]> {
    const orders = await executeApiRequest((apiClient) => apiClient.orders.getOrders());
    return orders.map(toOrder);
  },
  checkout(): Promise<ModelSwaggerMessageResponse> {
    return executeApiRequest((apiClient) => apiClient.orders.checkout());
  },
};
