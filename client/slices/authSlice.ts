import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface User {
  id: string;
  email: string;
  name: string;
  email_verified: boolean;
  is_admin: boolean;
}

interface AuthState {
  user: User | null;
  token: string | null;
}

function loadUser(): User | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem('user');
    return raw ? (JSON.parse(raw) as User) : null;
  } catch {
    return null;
  }
}

function loadToken(): string | null {
  if (typeof window === 'undefined') return null;
  try {
    return localStorage.getItem('auth_token');
  } catch {
    return null;
  }
}

const initialState: AuthState = {
  user: loadUser(),
  token: loadToken(),
};

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    /**
     * Called after a successful login or SSO. Persists both the user profile
     * and the raw JWT so the API client can attach it as an Authorization
     * header on every page load — even when the Go API is cross-origin and
     * the httpOnly cookie is not accessible to the Next.js server.
     *
     * Also writes a lightweight non-httpOnly `session` cookie so the Next.js
     * Edge middleware can detect an active session without needing access to
     * the httpOnly `jwt` cookie.
     */
    setCredentials(
      state,
      action: PayloadAction<{ user: User; token: string; expiresAt?: string }>
    ) {
      state.user = action.payload.user;
      state.token = action.payload.token;
      if (typeof window !== 'undefined') {
        localStorage.setItem('user', JSON.stringify(action.payload.user));
        localStorage.setItem('auth_token', action.payload.token);
        const maxAge = action.payload.expiresAt
          ? Math.max(
              0,
              Math.floor((new Date(action.payload.expiresAt).getTime() - Date.now()) / 1000)
            )
          : 86400;
        document.cookie = `session=1; path=/; SameSite=Lax; max-age=${maxAge}`;
      }
    },
    setUser(state, action: PayloadAction<User>) {
      state.user = action.payload;
      if (typeof window !== 'undefined') {
        localStorage.setItem('user', JSON.stringify(action.payload));
      }
    },
    clearCredentials(state) {
      state.user = null;
      state.token = null;
      if (typeof window !== 'undefined') {
        localStorage.removeItem('user');
        localStorage.removeItem('auth_token');
        document.cookie = 'session=; path=/; SameSite=Lax; max-age=-1';
      }
    },
  },
});

export const { setCredentials, setUser, clearCredentials } = authSlice.actions;
export default authSlice.reducer;
