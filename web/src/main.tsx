import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { store } from "./store";
import { theme } from "./theme";
import { App } from "./App";

const MSW_RELOAD_FLAG = "minitrace:reload-after-msw-unregister";

function isMockWorkerRegistration(
  registration: ServiceWorkerRegistration
): boolean {
  return [registration.active, registration.installing, registration.waiting].some(
    (worker) => worker?.scriptURL.includes("/mockServiceWorker.js") ?? false
  );
}

async function clearStaleMockServiceWorker(): Promise<boolean> {
  if (!("serviceWorker" in navigator)) {
    sessionStorage.removeItem(MSW_RELOAD_FLAG);
    return false;
  }

  const registrations = await navigator.serviceWorker.getRegistrations();
  const mockRegistrations = registrations.filter(isMockWorkerRegistration);
  if (mockRegistrations.length === 0) {
    sessionStorage.removeItem(MSW_RELOAD_FLAG);
    return false;
  }

  await Promise.all(mockRegistrations.map((registration) => registration.unregister()));

  const controlledByMock =
    navigator.serviceWorker.controller?.scriptURL.includes("/mockServiceWorker.js") ??
    false;
  if (controlledByMock && !sessionStorage.getItem(MSW_RELOAD_FLAG)) {
    sessionStorage.setItem(MSW_RELOAD_FLAG, "true");
    window.location.reload();
    return true;
  }

  sessionStorage.removeItem(MSW_RELOAD_FLAG);
  return false;
}

async function main() {
  // Keep dev pointed at the real Go backend unless mocks are explicitly requested.
  if (import.meta.env.DEV && import.meta.env.VITE_USE_MSW === "true") {
    const { worker } = await import("./mocks/browser");
    await worker.start({ onUnhandledRequest: "bypass" });
  } else if (await clearStaleMockServiceWorker()) {
    return;
  }

  createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <Provider store={store}>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <App />
        </ThemeProvider>
      </Provider>
    </StrictMode>
  );
}

main();
