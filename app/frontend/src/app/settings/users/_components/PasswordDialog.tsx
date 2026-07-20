"use client";

import * as React from "react";
import { useMutation } from "@tanstack/react-query";

import AdminService from "@/services/admin/admin";
import { UserWithRoles } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function PasswordDialog({
  open,
  onOpenChange,
  user,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: UserWithRoles | null;
}) {
  const [password, setPassword] = React.useState("");

  React.useEffect(() => {
    if (open) setPassword("");
  }, [open]);

  const mutation = useMutation({
    mutationFn: () => AdminService.setUserPassword(user!.id, password),
    onSuccess: () => {
      toast({
        variant: "success",
        title: "Password reset",
        description: "Every session for this user was signed out.",
      });
      onOpenChange(false);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't reset the password",
        description: apiErrorMessage(error),
      }),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Reset password for {user?.username}</DialogTitle>
          <DialogDescription>
            This signs the user out everywhere — a reset is usually a response to a compromise.
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            mutation.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="new_password">New password</Label>
            <Input
              id="new_password"
              type="password"
              value={password}
              required
              minLength={8}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending} showLoader={mutation.isPending}>
              Reset
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
