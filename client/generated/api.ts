/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface ModelCartItemResponse {
  added_at?: string;
  id?: string;
  listing?: ModelListingWithImages;
}

export interface ModelConversationResponse {
  last_message?: string;
  last_message_at?: string;
  listing_id?: string;
  listing_title?: string;
  other_user_id?: string;
  other_user_name?: string;
  unread_count?: number;
}

export interface ModelImageResponse {
  cdn_url?: string;
  id?: string;
}

export interface ModelListingWithImages {
  ai_confidence?: number;
  author?: string;
  condition?: string;
  course_code?: string;
  created_at?: string;
  department?: string;
  description?: string;
  id?: string;
  image_urls?: string[];
  isbn?: string;
  price?: number;
  seller_avatar?: string;
  seller_id?: string;
  seller_name?: string;
  status?: string;
  title?: string;
  updated_at?: string;
}

export interface ModelMessageResponse {
  body?: string;
  created_at?: string;
  id?: string;
  is_read?: boolean;
  listing_id?: string;
  receiver_id?: string;
  sender_id?: string;
  sender_name?: string;
}

export interface ModelNotificationResponse {
  created_at?: string;
  id?: string;
  is_read?: boolean;
  payload?: number[];
  type?: string;
}

export interface ModelOrderItemResponse {
  id?: string;
  listing?: ModelListingWithImages;
  price_at_purchase?: number;
}

export interface ModelOrderResponse {
  created_at?: string;
  id?: string;
  items?: ModelOrderItemResponse[];
  status?: string;
  total_amount?: number;
}

export interface ModelSwaggerAddToCartRequest {
  listing_id?: string;
}

export interface ModelSwaggerCreateListingRequest {
  author?: string;
  condition?: string;
  course_code?: string;
  department?: string;
  description?: string;
  image_ids?: string[];
  isbn?: string;
  price?: number;
  title?: string;
}

export interface ModelSwaggerErrorResponse {
  error?: string;
}

export interface ModelSwaggerLoginRequest {
  email?: string;
  password?: string;
}

export interface ModelSwaggerLoginResponse {
  expires_at?: string;
  token?: string;
  user?: ModelUser;
}

export interface ModelSwaggerMessageResponse {
  message?: string;
}

export interface ModelSwaggerRegisterImageRequest {
  cdn_url?: string;
  s3_key?: string;
}

export interface ModelSwaggerRegisterRequest {
  email?: string;
  name?: string;
  password?: string;
}

export interface ModelSwaggerResendVerificationRequest {
  email?: string;
}

export interface ModelSwaggerSendMessageRequest {
  body?: string;
}

export interface ModelSwaggerUpdateListingRequest {
  author?: string;
  condition?: string;
  course_code?: string;
  department?: string;
  description?: string;
  image_ids?: string[];
  isbn?: string;
  price?: number;
  status?: string;
  title?: string;
}

export interface ModelSwaggerUpdateUserRequest {
  avatar_image_id?: string;
  name?: string;
}

export interface ModelSwaggerUploadImageResponse {
  image?: ModelImageResponse;
}

export interface ModelUser {
  avatar_image_id?: string;
  cas_id?: string;
  created_at?: string;
  email?: string;
  email_verified?: boolean;
  id?: string;
  is_admin?: boolean;
  name?: string;
  updated_at?: string;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseFormat;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<RequestParams | void> | RequestParams | void;
  customFetch?: typeof fetch;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown>
  extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "/api/v1";
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private abortControllers = new Map<CancelToken, AbortController>();
  private customFetch = (...fetchParams: Parameters<typeof fetch>) =>
    fetch(...fetchParams);

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected encodeQueryParam(key: string, value: any) {
    const encodedKey = encodeURIComponent(key);
    return `${encodedKey}=${encodeURIComponent(typeof value === "number" ? value : `${value}`)}`;
  }

  protected addQueryParam(query: QueryParamsType, key: string) {
    return this.encodeQueryParam(key, query[key]);
  }

  protected addArrayQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];
    return value.map((v: any) => this.encodeQueryParam(key, v)).join("&");
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter(
      (key) => "undefined" !== typeof query[key],
    );
    return keys
      .map((key) =>
        Array.isArray(query[key])
          ? this.addArrayQueryParam(query, key)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.JsonApi]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.Text]: (input: any) =>
      input !== null && typeof input !== "string"
        ? JSON.stringify(input)
        : input,
    [ContentType.FormData]: (input: any) => {
      if (input instanceof FormData) {
        return input;
      }

      return Object.keys(input || {}).reduce((formData, key) => {
        const property = input[key];
        formData.append(
          key,
          property instanceof Blob
            ? property
            : typeof property === "object" && property !== null
              ? JSON.stringify(property)
              : `${property}`,
        );
        return formData;
      }, new FormData());
    },
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
  };

  protected mergeRequestParams(
    params1: RequestParams,
    params2?: RequestParams,
  ): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected createAbortSignal = (
    cancelToken: CancelToken,
  ): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = async <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format,
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.baseApiParams.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];
    const responseFormat = format || requestParams.format;

    return this.customFetch(
      `${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`,
      {
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type && type !== ContentType.FormData
            ? { "Content-Type": type }
            : {}),
        },
        signal:
          (cancelToken
            ? this.createAbortSignal(cancelToken)
            : requestParams.signal) || null,
        body:
          typeof body === "undefined" || body === null
            ? null
            : payloadFormatter(body),
      },
    ).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = null as unknown as T;
      r.error = null as unknown as E;

      const responseToParse = responseFormat ? response.clone() : response;
      const data = !responseFormat
        ? r
        : await responseToParse[responseFormat]()
            .then((data) => {
              if (r.ok) {
                r.data = data;
              } else {
                r.error = data;
              }
              return r;
            })
            .catch((e) => {
              r.error = e;
              return r;
            });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title NDHU Second-Hand Book Store API
 * @version 1.0
 * @license MIT
 * @termsOfService http://swagger.io/terms/
 * @baseUrl /api/v1
 * @contact SHBS Dev Team
 *
 * REST API for the NDHU campus second-hand textbook marketplace.
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  auth = {
    /**
     * @description Authenticates a user with email and password, returns a JWT token
     *
     * @tags Auth
     * @name LoginUser
     * @summary Login
     * @request POST:/auth/login
     */
    loginUser: (data: ModelSwaggerLoginRequest, params: RequestParams = {}) =>
      this.request<ModelSwaggerLoginResponse, ModelSwaggerErrorResponse>({
        path: `/auth/login`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Invalidates the current JWT and clears the session cookie
     *
     * @tags Auth
     * @name LogoutUser
     * @summary Logout
     * @request POST:/auth/logout
     * @secure
     */
    logoutUser: (params: RequestParams = {}) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/auth/logout`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Creates a new user account and sends an email verification link
     *
     * @tags Auth
     * @name RegisterUser
     * @summary Register
     * @request POST:/auth/register
     */
    registerUser: (
      data: ModelSwaggerRegisterRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/auth/register`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Sends a new verification email to the specified address if the account is unverified
     *
     * @tags Auth
     * @name ResendVerification
     * @summary Resend verification email
     * @request POST:/auth/resend-verification
     */
    resendVerification: (
      data: ModelSwaggerResendVerificationRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/auth/resend-verification`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Validates the CAS ticket and issues a JWT
     *
     * @tags Auth
     * @name SsoCallback
     * @summary SSO Callback
     * @request GET:/auth/sso/callback
     */
    ssoCallback: (
      query: {
        /** CAS ticket */
        ticket: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<any, void | ModelSwaggerErrorResponse>({
        path: `/auth/sso/callback`,
        method: "GET",
        query: query,
        ...params,
      }),

    /**
     * @description Redirects the browser to the NDHU CAS login page
     *
     * @tags Auth
     * @name SsoLogin
     * @summary SSO Login
     * @request GET:/auth/sso/login
     */
    ssoLogin: (params: RequestParams = {}) =>
      this.request<any, void | ModelSwaggerErrorResponse>({
        path: `/auth/sso/login`,
        method: "GET",
        ...params,
      }),

    /**
     * @description Verifies a user's email address using the token from the verification email
     *
     * @tags Auth
     * @name VerifyEmail
     * @summary Verify email
     * @request GET:/auth/verify
     */
    verifyEmail: (
      query: {
        /** Verification token */
        token: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/auth/verify`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),
  };
  cart = {
    /**
     * @description Returns the authenticated user's cart items
     *
     * @tags Cart
     * @name GetCart
     * @summary List cart
     * @request GET:/cart
     * @secure
     */
    getCart: (params: RequestParams = {}) =>
      this.request<ModelCartItemResponse[], ModelSwaggerErrorResponse>({
        path: `/cart`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Adds a listing to the authenticated user's cart
     *
     * @tags Cart
     * @name AddToCart
     * @summary Add to cart
     * @request POST:/cart
     * @secure
     */
    addToCart: (
      data: ModelSwaggerAddToCartRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/cart`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Removes a cart item from the authenticated user's cart
     *
     * @tags Cart
     * @name RemoveFromCart
     * @summary Remove from cart
     * @request DELETE:/cart/{id}
     * @secure
     */
    removeFromCart: (id: string, params: RequestParams = {}) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/cart/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),
  };
  images = {
    /**
     * @description Registers an image metadata record in the database after upload to S3
     *
     * @tags Images
     * @name RegisterImage
     * @summary Register image
     * @request POST:/images
     * @secure
     */
    registerImage: (
      data: ModelSwaggerRegisterImageRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelImageResponse, ModelSwaggerErrorResponse>({
        path: `/images`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Accepts a multipart file upload, stores it in S3, registers it in the database, and returns the image ID and CDN URL.
     *
     * @tags Images
     * @name UploadImage
     * @summary Upload image
     * @request POST:/images/upload
     * @secure
     */
    uploadImage: (
      data: {
        /**
         * Image file (JPEG, PNG, WebP; max 5 MB)
         * @format binary
         */
        file: File;
      },
      params: RequestParams = {},
    ) =>
      this.request<ModelSwaggerUploadImageResponse, ModelSwaggerErrorResponse>({
        path: `/images/upload`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.FormData,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns image metadata by ID
     *
     * @tags Images
     * @name GetImage
     * @summary Get image
     * @request GET:/images/{id}
     */
    getImage: (id: string, params: RequestParams = {}) =>
      this.request<ModelImageResponse, ModelSwaggerErrorResponse>({
        path: `/images/${id}`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  listings = {
    /**
     * @description Paginated, filterable list of book listings. Returns the full page in the JSON body and total count in X-Total-Count header.
     *
     * @tags Listings
     * @name GetListings
     * @summary List listings
     * @request GET:/listings
     */
    getListings: (
      query?: {
        /** Filter by status (default: active) */
        status?: string;
        /** Filter by department */
        department?: string;
        /** Filter by condition (good|moderate|poor) */
        condition?: string;
        /** Full-text search on title, author, or ISBN */
        search?: string;
        /** Minimum price */
        price_min?: number;
        /** Maximum price */
        price_max?: number;
        /** Page number (default: 1) */
        page?: number;
        /** Items per page (default: 20, max: 100) */
        limit?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<ModelListingWithImages[], ModelSwaggerErrorResponse>({
        path: `/listings`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description Publishes a new book listing for the authenticated seller.
     *
     * @tags Listings
     * @name CreateListing
     * @summary Create listing
     * @request POST:/listings
     * @secure
     */
    createListing: (
      data: ModelSwaggerCreateListingRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelListingWithImages, ModelSwaggerErrorResponse>({
        path: `/listings`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns all listings created by the authenticated user, newest first.
     *
     * @tags Listings
     * @name GetMyListings
     * @summary My listings
     * @request GET:/listings/me
     * @secure
     */
    getMyListings: (params: RequestParams = {}) =>
      this.request<ModelListingWithImages[], ModelSwaggerErrorResponse>({
        path: `/listings/me`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns a single listing with seller info and image URLs.
     *
     * @tags Listings
     * @name GetListing
     * @summary Get listing by ID
     * @request GET:/listings/{id}
     */
    getListing: (id: string, params: RequestParams = {}) =>
      this.request<ModelListingWithImages, ModelSwaggerErrorResponse>({
        path: `/listings/${id}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Soft-deletes a listing by setting its status to 'delisted'. Only the seller or an admin may delete.
     *
     * @tags Listings
     * @name DeleteListing
     * @summary Delete listing
     * @request DELETE:/listings/{id}
     * @secure
     */
    deleteListing: (id: string, params: RequestParams = {}) =>
      this.request<void, ModelSwaggerErrorResponse>({
        path: `/listings/${id}`,
        method: "DELETE",
        secure: true,
        ...params,
      }),

    /**
     * @description Partially updates a listing. Only the seller may update their own listing.
     *
     * @tags Listings
     * @name UpdateListing
     * @summary Update listing
     * @request PATCH:/listings/{id}
     * @secure
     */
    updateListing: (
      id: string,
      data: ModelSwaggerUpdateListingRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelListingWithImages, ModelSwaggerErrorResponse>({
        path: `/listings/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns the conversation between the authenticated user and the other party for a given listing. Marks received messages as read.
     *
     * @tags Messages
     * @name GetMessages
     * @summary List messages for a listing
     * @request GET:/listings/{listingId}/messages
     * @secure
     */
    getMessages: (listingId: string, params: RequestParams = {}) =>
      this.request<ModelMessageResponse[], ModelSwaggerErrorResponse>({
        path: `/listings/${listingId}/messages`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Sends a message from the authenticated user to the listing seller. Creates a notification for the receiver.
     *
     * @tags Messages
     * @name SendMessage
     * @summary Send a message about a listing
     * @request POST:/listings/{listingId}/messages
     * @secure
     */
    sendMessage: (
      listingId: string,
      data: ModelSwaggerSendMessageRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelMessageResponse, ModelSwaggerErrorResponse>({
        path: `/listings/${listingId}/messages`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  messages = {
    /**
     * @description Returns one summary entry per unique (listing, other_user) conversation pair the authenticated user is part of.
     *
     * @tags Messages
     * @name GetConversations
     * @summary List conversation threads
     * @request GET:/messages/conversations
     * @secure
     */
    getConversations: (params: RequestParams = {}) =>
      this.request<ModelConversationResponse[], ModelSwaggerErrorResponse>({
        path: `/messages/conversations`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns the number of unread messages for the authenticated user.
     *
     * @tags Messages
     * @name GetUnreadMessageCount
     * @summary Unread message count
     * @request GET:/messages/unread-count
     * @secure
     */
    getUnreadMessageCount: (params: RequestParams = {}) =>
      this.request<Record<string, number>, ModelSwaggerErrorResponse>({
        path: `/messages/unread-count`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Marks a single message as read. Only the receiver may mark a message as read.
     *
     * @tags Messages
     * @name MarkMessageAsRead
     * @summary Mark message as read
     * @request PATCH:/messages/{id}/read
     * @secure
     */
    markMessageAsRead: (id: string, params: RequestParams = {}) =>
      this.request<void, ModelSwaggerErrorResponse>({
        path: `/messages/${id}/read`,
        method: "PATCH",
        secure: true,
        ...params,
      }),
  };
  notifications = {
    /**
     * @description Returns all notifications for the authenticated user, newest first.
     *
     * @tags Notifications
     * @name GetNotifications
     * @summary List notifications
     * @request GET:/notifications
     * @secure
     */
    getNotifications: (params: RequestParams = {}) =>
      this.request<ModelNotificationResponse[], ModelSwaggerErrorResponse>({
        path: `/notifications`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Marks every unread notification for the authenticated user as read.
     *
     * @tags Notifications
     * @name MarkAllNotificationsAsRead
     * @summary Mark all notifications as read
     * @request PATCH:/notifications/read-all
     * @secure
     */
    markAllNotificationsAsRead: (params: RequestParams = {}) =>
      this.request<void, ModelSwaggerErrorResponse>({
        path: `/notifications/read-all`,
        method: "PATCH",
        secure: true,
        ...params,
      }),

    /**
     * @description Returns the number of unread notifications for the authenticated user.
     *
     * @tags Notifications
     * @name GetUnreadNotificationCount
     * @summary Unread notification count
     * @request GET:/notifications/unread-count
     * @secure
     */
    getUnreadNotificationCount: (params: RequestParams = {}) =>
      this.request<Record<string, number>, ModelSwaggerErrorResponse>({
        path: `/notifications/unread-count`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Marks a single notification as read. Returns 404 if the notification does not belong to the current user.
     *
     * @tags Notifications
     * @name MarkNotificationAsRead
     * @summary Mark notification as read
     * @request PATCH:/notifications/{id}/read
     * @secure
     */
    markNotificationAsRead: (id: string, params: RequestParams = {}) =>
      this.request<void, ModelSwaggerErrorResponse>({
        path: `/notifications/${id}/read`,
        method: "PATCH",
        secure: true,
        ...params,
      }),
  };
  orders = {
    /**
     * @description Returns all orders for the authenticated user, newest first
     *
     * @tags Orders
     * @name GetOrders
     * @summary List orders
     * @request GET:/orders
     * @secure
     */
    getOrders: (params: RequestParams = {}) =>
      this.request<ModelOrderResponse[], ModelSwaggerErrorResponse>({
        path: `/orders`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Creates an order from the authenticated user's cart, marks listings as sold, and clears the cart
     *
     * @tags Orders
     * @name Checkout
     * @summary Checkout
     * @request POST:/orders
     * @secure
     */
    checkout: (params: RequestParams = {}) =>
      this.request<ModelSwaggerMessageResponse, ModelSwaggerErrorResponse>({
        path: `/orders`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),
  };
  users = {
    /**
     * @description Returns the authenticated user's profile information
     *
     * @tags Users
     * @name GetMe
     * @summary Get current user
     * @request GET:/users/me
     * @secure
     */
    getMe: (params: RequestParams = {}) =>
      this.request<ModelUser, ModelSwaggerErrorResponse>({
        path: `/users/me`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Updates the authenticated user's name and/or avatar image
     *
     * @tags Users
     * @name UpdateMe
     * @summary Update current user
     * @request PUT:/users/me
     * @secure
     */
    updateMe: (
      data: ModelSwaggerUpdateUserRequest,
      params: RequestParams = {},
    ) =>
      this.request<ModelUser, ModelSwaggerErrorResponse>({
        path: `/users/me`,
        method: "PUT",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
}
