import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface CartItem {
  listingId: string;
  title: string;
  price: number;
  imageUrl: string;
  sellerName: string;
}

interface CartState {
  items: CartItem[];
}

const initialState: CartState = {
  items: [],
};

const cartSlice = createSlice({
  name: 'cart',
  initialState,
  reducers: {
    setCart(state, action: PayloadAction<CartItem[]>) {
      state.items = action.payload;
    },
    addItem(state, action: PayloadAction<CartItem>) {
      const exists = state.items.find((i) => i.listingId === action.payload.listingId);
      if (!exists) state.items.push(action.payload);
    },
    removeItem(state, action: PayloadAction<string>) {
      state.items = state.items.filter((i) => i.listingId !== action.payload);
    },
    clearCart(state) {
      state.items = [];
    },
  },
});

export const { setCart, addItem, removeItem, clearCart } = cartSlice.actions;
export default cartSlice.reducer;
