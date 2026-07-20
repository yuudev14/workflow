"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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

        {/* SSO providers (Keycloak, Google) get listed here in M3: fetch
            /auth/v1/providers and render a button per provider. */}
      </div>
    </div>
  );
}
