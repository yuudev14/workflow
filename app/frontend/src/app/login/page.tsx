"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { AxiosError } from "axios";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import AuthService from "@/services/auth/auth";
import { ME_QUERY_KEY } from "@/components/provider/auth-provider";

const loginSchema = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);
  const [ssoError, setSsoError] = useState(false);

  // The OIDC callback bounces here with ?error=sso when a round trip fails.
  // Read it off the URL rather than useSearchParams to avoid the Suspense
  // requirement that would otherwise fail the production build.
  useEffect(() => {
    setSsoError(new URLSearchParams(window.location.search).get("error") === "sso");
  }, []);

  const providersQuery = useQuery({
    queryKey: ["auth-providers"],
    queryFn: AuthService.getProviders,
  });
  const providers = providersQuery.data ?? [];

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (values: LoginForm) => {
    setFormError(null);
    try {
      await AuthService.login(values);
      // The session cookies are set; drop any stale /me so the shell renders
      // with the user that just signed in.
      await queryClient.invalidateQueries({ queryKey: ME_QUERY_KEY });
      router.replace("/");
    } catch (err) {
      // The API answers every failure the same way on purpose, so there is
      // nothing more specific to show.
      const status = (err as AxiosError)?.response?.status;
      setFormError(
        status === 401
          ? "Incorrect username or password."
          : "Could not sign in. Please try again.",
      );
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3">
          <span className="flex aspect-square size-10 items-center justify-center rounded-md bg-foreground text-[16px] font-bold text-background">
            Y
          </span>
          <div className="text-center">
            <h1 className="text-lg font-semibold">Sign in to YTSoar</h1>
            <p className="text-sm text-muted-foreground">SOAR platform</p>
          </div>
        </div>

        {ssoError && (
          <p className="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
            Single sign-on didn't complete. Try again, or use your username and password.
          </p>
        )}

        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col gap-4 rounded-lg border bg-card p-6"
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="username">Username</Label>
            <Input
              id="username"
              autoComplete="username"
              autoFocus
              {...register("username")}
            />
            {errors.username && (
              <p className="text-xs text-destructive">{errors.username.message}</p>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              {...register("password")}
            />
            {errors.password && (
              <p className="text-xs text-destructive">{errors.password.message}</p>
            )}
          </div>

          {formError && (
            <p className="text-sm text-destructive" role="alert">
              {formError}
            </p>
          )}

          <Button type="submit" disabled={isSubmitting} className="w-full">
            {isSubmitting ? "Signing in…" : "Sign in"}
          </Button>
        </form>

        {providers.length > 0 && (
          <div className="mt-5">
            <div className="mb-4 flex items-center gap-3 text-xs uppercase tracking-wide text-muted-foreground">
              <span className="h-px flex-1 bg-border" />
              or
              <span className="h-px flex-1 bg-border" />
            </div>
            <div className="flex flex-col gap-2">
              {providers.map((provider) => (
                <Button
                  key={provider.id}
                  type="button"
                  variant="outline"
                  className="w-full"
                  // A full-page navigation, not fetch: the IdP round trip sets
                  // cookies, which only a top-level request can carry back.
                  onClick={() => window.location.assign(provider.start_url)}
                >
                  Continue with {provider.name}
                </Button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
