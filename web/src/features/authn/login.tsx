import { Logo } from "@components/logo";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { Alert, AlertDescription } from "@components/ui/alert";

export function LoginPage() {
  const failed = new URLSearchParams(window.location.search).has("sso_error");
  return (
    <div className="flex min-h-dvh w-full min-w-0 items-center justify-center bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center justify-items-center gap-3 text-center">
          <div className="flex items-center gap-2">
            <Logo className="text-primary" />
            <span className="text-lg font-semibold tracking-tight">Sigweft</span>
          </div>
          <CardTitle>Log In</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {failed && (
            <Alert variant="destructive">
              <AlertDescription>
                Could not sign in. Try again or check your access with your administrator.
              </AlertDescription>
            </Alert>
          )}
          <Button render={<a href="/api/auth/sso/start">Log In with SSO</a>} nativeButton={false} />
        </CardContent>
      </Card>
    </div>
  );
}
