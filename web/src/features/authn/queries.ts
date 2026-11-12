import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deleteSession, getSession } from "@lib/api-client/sdk.gen";
import type { Session } from "@lib/api-client/types.gen";
import { sessionTransport } from "@lib/session";

export const sessionQueryOptions = queryOptions({
  queryKey: ["auth", "session"],
  queryFn: async ({ signal }) => (await getSession({ signal, throwOnError: true })).data,
});

export function useSession() {
  return useQuery(sessionQueryOptions);
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => deleteSession({ throwOnError: true }),
    onSuccess: () => {
      sessionTransport.renew();
      queryClient.setQueryData<Session>(sessionQueryOptions.queryKey, { sso_enabled: true });
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== "auth" });
    },
  });
}
