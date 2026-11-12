import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Editor } from "@components/editor";
import { Logo } from "@components/logo";
import { Alert, AlertDescription, AlertTitle } from "@components/ui/alert";
import { Button } from "@components/ui/button";
import { LoginPage } from "@features/authn/login";
import { useLogout, useSession } from "@features/authn/queries";
import { getForm } from "@lib/api-client/sdk.gen";
import { draftKey } from "@lib/draft";

export function App() {
  const session = useSession();
  const logout = useLogout();
  const user = session.data?.user;
  const allowed = !!session.data && (!session.data.sso_enabled || !!user);
  const query = useQuery({
    queryKey: ["form"],
    queryFn: async ({ signal }) => (await getForm({ signal, throwOnError: true })).data,
    enabled: allowed,
  });
  useEffect(() => {
    if (allowed && window.location.pathname === "/login")
      window.history.replaceState(null, "", "/");
  }, [allowed]);
  if (session.data?.sso_enabled && !user) return <LoginPage />;
  return (
    <div className="min-h-screen">
      <header className="border-b bg-card">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-6 py-5 lg:px-10">
          <div className="flex items-center gap-2">
            <Logo className="text-primary" />
            <span className="text-lg font-semibold tracking-tight">Sigweft</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-sm text-muted-foreground">
              {user?.name || query.data?.title || "Email signatures"}
            </span>
            {user && (
              <Button
                variant="outline"
                size="sm"
                disabled={logout.isPending}
                onClick={() => logout.mutate()}
              >
                Log Out
              </Button>
            )}
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-9 lg:px-10 lg:py-12">
        {logout.isError && (
          <Alert variant="destructive" className="mb-6">
            <AlertDescription>Could not sign out. Try again.</AlertDescription>
          </Alert>
        )}
        {allowed && query.data ? (
          <Editor key={draftKey(query.data, user)} metadata={query.data} user={user} />
        ) : session.isError || query.isError ? (
          <Alert variant="destructive">
            <AlertTitle>Could not load the form</AlertTitle>
            <AlertDescription>
              <p>Check your connection and try again.</p>
              <Button
                variant="outline"
                onClick={() => void (session.isError ? session.refetch() : query.refetch())}
              >
                Try Again
              </Button>
            </AlertDescription>
          </Alert>
        ) : (
          <output className="text-muted-foreground">Loading the signature form…</output>
        )}
      </main>
    </div>
  );
}
