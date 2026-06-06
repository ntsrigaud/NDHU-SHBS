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
