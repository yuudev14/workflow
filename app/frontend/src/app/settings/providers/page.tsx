"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { KeyRound, Pencil, Plus } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { AuthProviderAdmin } from "@/services/admin/admin.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import {
  DataTable,
  EmptyState,
  Glyph,
  PageShell,
  StatusPill,
} from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Chip } from "../_components/Chip";

// Per-IdP config presets. The template `kind` is a UI convenience only — it
// seeds the config editor below; the backend stores just type=oidc and runs one
// generic OIDC engine (see docker/keycloak/README.md). Google's token carries no
// groups, so it leans on default_role; Azure/Okta emit role names in a claim, so
// the mapping table is optional (unmatched values pass through).
const OIDC_PRESETS: Record<string, { label: string; config: Record<string, unknown> }> = {
  keycloak: {
    label: "Keycloak",
    config: {
      issuer: "http://localhost:8180/realms/ytsoar",
      internal_issuer: "http://ytsoar_keycloak:8180/realms/ytsoar",
      client_id: "ytsoar",
      client_secret: "",
      scopes: ["openid", "profile", "email"],
      groups_claim: "groups",
      group_role_mapping: { "soc-admins": "admin", "soc-analysts": "analyst" },
      default_role: "viewer",
      sync_mode: "roles",
      allow_jit: true,
    },
  },
  google: {
    label: "Google Workspace",
    config: {
      issuer: "https://accounts.google.com",
      client_id: "",
      client_secret: "",
      scopes: ["openid", "email", "profile"],
      default_role: "viewer",
      sync_mode: "attributes",
      allow_jit: true,
    },
  },
  okta: {
    label: "Okta",
    config: {
      issuer: "https://your-org.okta.com",
      client_id: "",
      client_secret: "",
      scopes: ["openid", "profile", "email", "groups"],
      groups_claim: "groups",
      group_role_mapping: { "SOC Admins": "admin", "SOC Analysts": "analyst" },
      default_role: "viewer",
      sync_mode: "roles",
      allow_jit: true,
    },
  },
  azure: {
    label: "Microsoft Entra (Azure AD)",
    config: {
      issuer: "https://login.microsoftonline.com/<tenant-id>/v2.0",
      client_id: "",
      client_secret: "",
      scopes: ["openid", "profile", "email"],
      groups_claim: "roles",
      default_role: "viewer",
      sync_mode: "roles",
      allow_jit: true,
    },
  },
  generic: {
    label: "Generic OIDC",
    config: {
      issuer: "https://idp.example.com",
      client_id: "",
      client_secret: "",
      scopes: ["openid", "profile", "email"],
      groups_claim: "groups",
      group_role_mapping: {},
      default_role: "viewer",
      sync_mode: "roles",
      allow_jit: true,
    },
  },
};

const DEFAULT_PRESET = "keycloak";

const columnHelper = createColumnHelper<AuthProviderAdmin>();

export default function ProvidersPage() {
  const canCreate = usePermission("settings", "create");
  const canUpdate = usePermission("settings", "update");

  const [editing, setEditing] = React.useState<AuthProviderAdmin | null>(null);
  const [open, setOpen] = React.useState(false);

  const providersQuery = useQuery({
    queryKey: ["auth-providers-admin"],
    queryFn: AdminService.listAuthProviders,
  });
  const providers = providersQuery.data ?? [];

  const openDialog = (provider: AuthProviderAdmin | null) => {
    setEditing(provider);
    setOpen(true);
  };

  const columns = React.useMemo(
    () => [
      columnHelper.accessor("name", {
        header: "Provider",
        cell: ({ row: { original: p } }) => (
          <div className="flex items-center gap-2.5">
            <Glyph icon={KeyRound} tone={p.enabled ? "signal" : "slate"} />
            <div className="font-semibold">{p.name}</div>
          </div>
        ),
      }),
      columnHelper.accessor("type", {
        header: "Type",
        cell: ({ getValue }) => <Chip>{getValue().toUpperCase()}</Chip>,
      }),
      columnHelper.accessor("enabled", {
        header: "Status",
        cell: ({ getValue }) => (
          <StatusPill variant={getValue() ? "success" : "neutral"}>
            {getValue() ? "Enabled" : "Disabled"}
          </StatusPill>
        ),
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        meta: { align: "right" },
        cell: ({ row: { original: p } }) =>
          canUpdate && (
            <div className="flex justify-end">
              <Button variant="ghost" size="icon" title="Edit" onClick={() => openDialog(p)}>
                <Pencil />
              </Button>
            </div>
          ),
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canUpdate]
  );

  return (
    <PageShell
      title="Providers"
      subtitle="Single sign-on sources. Users signing in through one are provisioned and role-synced automatically."
      actions={
        canCreate && (
          <Button onClick={() => openDialog(null)}>
            <Plus /> New provider
          </Button>
        )
      }
    >
      {providersQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-md" />
          ))}
        </div>
      ) : providers.length === 0 ? (
        <EmptyState
          icon={KeyRound}
          title="No providers"
          description="Add a Keycloak or other OIDC provider to enable single sign-on."
        />
      ) : (
        <DataTable columns={columns} data={providers} getRowId={(p) => p.id} pageSize={10} />
      )}

      <ProviderDialog open={open} onOpenChange={setOpen} provider={editing} />
    </PageShell>
  );
}

function ProviderDialog({
  open,
  onOpenChange,
  provider,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  provider: AuthProviderAdmin | null;
}) {
  const queryClient = useQueryClient();
  const editing = provider !== null;

  const [name, setName] = React.useState("");
  const [type, setType] = React.useState("oidc");
  const [enabled, setEnabled] = React.useState(true);
  const [preset, setPreset] = React.useState(DEFAULT_PRESET);
  const [configText, setConfigText] = React.useState("");
  const [configError, setConfigError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setConfigError(null);
    if (provider) {
      setName(provider.name);
      setType(provider.type);
      setEnabled(provider.enabled);
      setConfigText(JSON.stringify(provider.config, null, 2));
    } else {
      setName("");
      setType("oidc");
      setEnabled(true);
      setPreset(DEFAULT_PRESET);
      setConfigText(JSON.stringify(OIDC_PRESETS[DEFAULT_PRESET].config, null, 2));
    }
  }, [open, provider]);

  const applyPreset = (kind: string) => {
    setPreset(kind);
    setConfigText(JSON.stringify(OIDC_PRESETS[kind].config, null, 2));
    setConfigError(null);
  };

  const save = useMutation({
    mutationFn: async () => {
      let config: Record<string, unknown>;
      try {
        config = JSON.parse(configText);
      } catch {
        throw new Error("Config is not valid JSON.");
      }
      if (editing) {
        await AdminService.updateAuthProvider(provider.id, { name, config, enabled });
      } else {
        await AdminService.createAuthProvider({ type, name, config, enabled });
      }
    },
    onSuccess: () => {
      toast({ variant: "success", title: editing ? "Provider updated" : "Provider created" });
      queryClient.invalidateQueries({ queryKey: ["auth-providers-admin"] });
      onOpenChange(false);
    },
    onError: (error) =>
      toast({
        variant: "destructive",
        title: "Couldn't save the provider",
        description: error instanceof Error ? error.message : apiErrorMessage(error),
      }),
  });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    try {
      JSON.parse(configText);
      setConfigError(null);
    } catch {
      setConfigError("Config is not valid JSON.");
      return;
    }
    save.mutate();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[calc(100dvh-3rem)] max-w-2xl flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="border-b border-line px-6 pt-6 pb-4">
          <DialogTitle>{editing ? `Edit ${provider.name}` : "New provider"}</DialogTitle>
          <DialogDescription>
            Secrets are write-only — the stored value is kept if you leave the masked
            <code className="px-1">********</code> in place.
          </DialogDescription>
        </DialogHeader>

        <form className="flex min-h-0 flex-1 flex-col" onSubmit={submit}>
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-6 py-5">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="provider_name">Name</Label>
              <Input
                id="provider_name"
                value={name}
                required
                onChange={(e) => setName(e.target.value)}
                placeholder="Keycloak"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="provider_type">Type</Label>
              <Select value={type} onValueChange={setType} disabled={editing}>
                <SelectTrigger id="provider_type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="oidc">OIDC (Keycloak, Google, …)</SelectItem>
                  <SelectItem value="ldap">LDAP</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <label className="flex items-center gap-2.5 text-[13.5px]">
              <input
                type="checkbox"
                className="size-4 accent-primary"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              Enabled — appears on the login screen
            </label>

            {!editing && type === "oidc" && (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="provider_preset">Template</Label>
                <Select value={preset} onValueChange={applyPreset}>
                  <SelectTrigger id="provider_preset">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(OIDC_PRESETS).map(([kind, p]) => (
                      <SelectItem key={kind} value={kind}>
                        {p.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span className="text-[12px] text-ink-faint">
                  Seeds the config below — a starting point you can edit. Backend stores the type only.
                </span>
              </div>
            )}

            <div className="flex flex-col gap-1.5">
              <div className="flex items-baseline justify-between">
                <Label htmlFor="provider_config">Config (JSON)</Label>
                <span className="text-[12px] text-ink-faint">
                  Provider settings only — issuer, client_id, … (not name/type)
                </span>
              </div>
              <Textarea
                id="provider_config"
                value={configText}
                onChange={(e) => setConfigText(e.target.value)}
                spellCheck={false}
                className="min-h-64 font-mono text-[12.5px]"
              />
              {configError && <p className="text-xs text-destructive">{configError}</p>}
            </div>
          </div>

          <DialogFooter className="border-t border-line px-6 py-4">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={save.isPending} showLoader={save.isPending}>
              {editing ? "Save" : "Create provider"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
