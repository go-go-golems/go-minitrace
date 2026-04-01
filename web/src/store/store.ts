import { configureStore } from "@reduxjs/toolkit";
import { minitraceApi } from "../api/minitrace";
import { uiReducer } from "./uiSlice";

export const store = configureStore({
  reducer: {
    [minitraceApi.reducerPath]: minitraceApi.reducer,
    ui: uiReducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(minitraceApi.middleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
