import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { client as apiClient } from "@lib/api-client/client.gen";
import type { Session } from "@lib/api-client/types.gen";
import { sessionTransport } from "@lib/session";
import { sessionQueryOptions } from "@features/authn/queries";
import { App } from "./app";
import "./index.css";

const client = new QueryClient({
  defaultOptions: { queries: { refetchOnWindowFocus: false, retry: 1 } },
});

apiClient.setConfig({ fetch: sessionTransport.fetch });
sessionTransport.onExpired(() => {
  client.setQueryData<Session>(sessionQueryOptions.queryKey, { sso_enabled: true });
  client.removeQueries({ predicate: (query) => query.queryKey[0] !== "auth" });
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
