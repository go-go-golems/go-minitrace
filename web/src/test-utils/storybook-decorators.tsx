import type { Decorator } from "@storybook/react-vite";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { Provider } from "react-redux";
import { configureStore } from "@reduxjs/toolkit";
import { theme } from "../theme";
import { minitraceApi } from "../api/minitrace";
import { uiReducer } from "../store";

/** MUI theme decorator */
export const withTheme: Decorator = (Story) => (
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <Story />
  </ThemeProvider>
);

/** Redux store decorator (fresh store per story) */
export const withStore: Decorator = (Story) => {
  const store = configureStore({
    reducer: {
      [minitraceApi.reducerPath]: minitraceApi.reducer,
      ui: uiReducer,
    },
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(minitraceApi.middleware),
  });
  return (
    <Provider store={store}>
      <Story />
    </Provider>
  );
};

/** Combined decorator for all stories */
export const withAll: Decorator = (Story) => {
  const store = configureStore({
    reducer: {
      [minitraceApi.reducerPath]: minitraceApi.reducer,
      ui: uiReducer,
    },
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(minitraceApi.middleware),
  });
  return (
    <Provider store={store}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Story />
      </ThemeProvider>
    </Provider>
  );
};
