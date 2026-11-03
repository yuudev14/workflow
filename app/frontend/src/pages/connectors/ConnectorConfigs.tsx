import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Pencil, Plus, Trash2 } from "lucide-react";

import ConnectorService from "@/services/connectors/connectors";
import { ConnectorConfig, ConnectorInfo, Field } from "@/services/connectors/connectors.schema";
import { apiErrorMessage } from "@/services/common/errors";
import { toast } from "@/hooks/use-toast";
import { usePermission } from "@/hooks/usePermission";
import { EmptyState, StatusPill } from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/pages/settings/components/ConfirmDialog";

export const configsQueryKey = (connectorId: string) => ["connector-configs", connectorId];

// The Configurations tab of a connector: named configs, with the form built from info.json.
export function ConnectorConfigs({ connector }: { connector: ConnectorInfo }) {
  const canUpdate = usePermission("connectors", "update");
  const queryClient = useQueryClient();
  const [editing, setEditing] = React.useState<ConnectorConfig | null>(null);
  const [open, setOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState<ConnectorConfig | null>(null);

  const configsQuery = useQuery({
    queryKey: configsQueryKey(connector.id),
    queryFn: () => ConnectorService.listConfigs(connector.id),
  });
  const configs = configsQuery.data ?? [];

  const remove = useMutation({
    mutationFn: (config: ConnectorConfig) => ConnectorService.deleteConfig(connector.id, config.id),
    onSuccess: () => {
      toast({ variant: "success", title: "Configuration deleted" });
      queryClient.invalidateQueries({ queryKey: configsQueryKey(connector.id) });
      setDeleting(null);
    },
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't delete the configuration", description: apiErrorMessage(error) }),
  });

  const openDialog = (config: ConnectorConfig | null) => {
    setEditing(config);
    setOpen(true);
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-[13px] text-ink-soft">
          Values are stored encrypted. A step with no configuration uses the default.
        </p>
        {canUpdate && (
          <Button size="sm" onClick={() => openDialog(null)}>
            <Plus /> New configuration
          </Button>
        )}
      </div>

      {configsQuery.isLoading ? (
        <Skeleton className="h-14 rounded-md" />
      ) : configs.length === 0 ? (
        <EmptyState icon={KeyRound} title="No configurations" description="Add one to give this connector its URL and keys." />
      ) : (
        <ul className="flex flex-col divide-y divide-line rounded-md border border-line">
          {configs.map((config) => (
            <li key={config.id} className="flex items-center justify-between gap-2 px-3 py-2.5">
              <div className="flex items-center gap-2">
                <span className="text-[14px] font-semibold">{config.name}</span>
                {config.is_default && <StatusPill variant="success">Default</StatusPill>}
              </div>
              {canUpdate && (
                <div className="flex">
                  <Button variant="ghost" size="icon" title={`Edit ${config.name}`} onClick={() => openDialog(config)}>
                    <Pencil />
                  </Button>
                  <Button variant="ghost" size="icon" title={`Delete ${config.name}`} onClick={() => setDeleting(config)}>
                    <Trash2 />
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      <ConfigDialog open={open} onOpenChange={setOpen} connector={connector} config={editing} />
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(next) => !next && setDeleting(null)}
        title="Delete the configuration?"
        description={`Steps that use ${deleting?.name} fail until you pick another configuration.`}
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => deleting && remove.mutate(deleting)}
      />
    </div>
  );
}

function ConfigDialog({
  open,
  onOpenChange,
  connector,
  config,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  connector: ConnectorInfo;
  config: ConnectorConfig | null;
}) {
  const queryClient = useQueryClient();
  const editing = config !== null;
  const fields = (connector.configuration?.fields ?? []).filter((f) => f.visible !== false);

  const [name, setName] = React.useState("");
  const [isDefault, setIsDefault] = React.useState(false);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [values, setValues] = React.useState<Record<string, any>>({});

  React.useEffect(() => {
    if (!open) return;
    setName(config?.name ?? "");
    setIsDefault(config?.is_default ?? false);
    setValues(config ? { ...config.config } : Object.fromEntries(fields.map((f) => [f.name, f.value ?? (f.type === "checkbox" ? false : "")])));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, config]);

  const save = useMutation({
    mutationFn: () => {
      const input = { name, config: values, ...(isDefault ? { is_default: true } : {}) };
      return editing
        ? ConnectorService.updateConfig(connector.id, config.id, input)
        : ConnectorService.createConfig(connector.id, input);
    },
    onSuccess: () => {
      toast({ variant: "success", title: editing ? "Configuration updated" : "Configuration created" });
      queryClient.invalidateQueries({ queryKey: configsQueryKey(connector.id) });
      onOpenChange(false);
    },
    onError: (error) =>
      toast({ variant: "destructive", title: "Couldn't save the configuration", description: apiErrorMessage(error) }),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `Edit ${config.name}` : "New configuration"}</DialogTitle>
          <DialogDescription>
            Secrets are write-only. Leave <code className="px-1">********</code> in place to keep the stored value.
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            save.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="config_name">Name</Label>
            <Input id="config_name" value={name} required maxLength={50} placeholder="prod" onChange={(e) => setName(e.target.value)} />
          </div>
          {fields.map((field) => (
            <ConfigField
              key={field.name}
              field={field}
              value={values[field.name]}
              onChange={(value) => setValues((prev) => ({ ...prev, [field.name]: value }))}
            />
          ))}
          {!config?.is_default && (
            <label className="flex items-center gap-2.5 text-[13.5px]">
              <input type="checkbox" className="size-4 accent-primary" checked={isDefault} onChange={(e) => setIsDefault(e.target.checked)} />
              Default configuration
            </label>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={save.isPending} showLoader={save.isPending}>
              {editing ? "Save" : "Create configuration"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function ConfigField({
  field,
  value,
  onChange,
}: {
  field: Field;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  onChange: (value: any) => void;
}) {
  const id = `config_field_${field.name}`;
  if (field.type === "checkbox") {
    return (
      <label className="flex items-center gap-2.5 text-[13.5px]" title={field.tooltip}>
        <input id={id} type="checkbox" className="size-4 accent-primary" checked={Boolean(value)} onChange={(e) => onChange(e.target.checked)} />
        {field.title}
      </label>
    );
  }
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{field.title}</Label>
      <Input
        id={id}
        type={field.type === "password" ? "password" : "text"}
        autoComplete="off"
        required={field.required}
        disabled={field.editable === false}
        title={field.tooltip}
        value={value ?? ""}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
