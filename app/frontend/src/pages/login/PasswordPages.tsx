import { Link } from "react-router";
import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import AuthService from "@/services/auth/auth";
import { apiErrorMessage } from "@/services/common/errors";
import { AuthLayout } from "./LoginPage";

const backToSignIn = (
  <Link to="/login" className="mt-5 block text-center text-sm text-muted-foreground hover:text-foreground">
    Back to sign in
  </Link>
);

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const mutation = useMutation({ mutationFn: () => AuthService.forgotPassword(email) });

  return (
    <AuthLayout title="Forgot your password?" subtitle="We email you a link to set a new one.">
      {mutation.isSuccess ? (
        <p className="rounded-lg border bg-card p-6 text-sm" role="status">
          If an account uses {email}, it gets an email with a link. The link works for 1 hour.
        </p>
      ) : (
        <form
          className="flex flex-col gap-4 rounded-lg border bg-card p-6"
          onSubmit={(e) => {
            e.preventDefault();
            mutation.mutate();
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="email">Email</Label>
            <Input id="email" type="email" autoComplete="email" autoFocus required value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          {mutation.isError && (
            <p className="text-sm text-destructive" role="alert">
              {apiErrorMessage(mutation.error)}
            </p>
          )}
          <Button type="submit" disabled={mutation.isPending} className="w-full">
            Send link
          </Button>
        </form>
      )}
      {backToSignIn}
    </AuthLayout>
  );
}

/** Opened from an invite email or a reset email. Both links carry ?token=. */
export function SetPasswordPage() {
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const mismatch = confirm !== "" && password !== confirm;

  // Read off the URL, as the login page does with ?error=sso
  useEffect(() => {
    setToken(new URLSearchParams(window.location.search).get("token") ?? "");
  }, []);

  const mutation = useMutation({ mutationFn: () => AuthService.resetPassword(token, password) });

  return (
    <AuthLayout title="Set your password" subtitle="Choose a password with at least 8 characters.">
      {mutation.isSuccess ? (
        <p className="rounded-lg border bg-card p-6 text-sm" role="status">
          Your password is set. Sign in with your username and the new password.
        </p>
      ) : (
        <form
          className="flex flex-col gap-4 rounded-lg border bg-card p-6"
          onSubmit={(e) => {
            e.preventDefault();
            if (!mismatch) mutation.mutate();
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="password">New password</Label>
            <Input id="password" type="password" autoComplete="new-password" autoFocus required minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="confirm">Confirm password</Label>
            <Input id="confirm" type="password" autoComplete="new-password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} />
            {mismatch && <p className="text-xs text-destructive">The passwords do not match.</p>}
          </div>
          {(mutation.isError || !token) && (
            <p className="text-sm text-destructive" role="alert">
              {token ? apiErrorMessage(mutation.error) : "The link has no token. Open the link from the email again."}
            </p>
          )}
          <Button type="submit" disabled={mutation.isPending || !token} className="w-full">
            Set password
          </Button>
        </form>
      )}
      {backToSignIn}
    </AuthLayout>
  );
}
