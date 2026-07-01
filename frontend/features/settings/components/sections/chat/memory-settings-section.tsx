"use client";

import * as React from "react";
import { Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { MEMORY_LIMIT } from "@/features/settings/model/memory-settings";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { deleteUserMemoryByID, listUserMemories } from "@/shared/api/memory";
import type { UserMemoryDTO } from "@/shared/api/memory.types";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import {
  SettingsFieldList,
  SettingsFieldRow,
  SettingsSection,
} from "@/shared/components/settings-layout";

type MemorySettingsSectionProps = {
  enabled: boolean;
  loading: boolean;
  onEnabledChange: (enabled: boolean) => void;
};

export function MemorySettingsSection({ enabled, loading, onEnabledChange }: MemorySettingsSectionProps) {
  const t = useTranslations("settings.chatPage.memory");
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [items, setItems] = React.useState<UserMemoryDTO[]>([]);
  const [loadingMemories, setLoadingMemories] = React.useState(true);
  const [selectedMemory, setSelectedMemory] = React.useState<UserMemoryDTO | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<UserMemoryDTO | null>(null);
  const [deleting, setDeleting] = React.useState(false);

  React.useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoadingMemories(true);
      try {
        const token = await resolveAccessToken();
        if (!token) return;
        const memories = await listUserMemories(token);
        if (!cancelled) setItems(memories);
      } catch (error) {
        if (!cancelled) toast.error(t("loadFailed"), { description: resolveErrorMessage(error) });
      } finally {
        if (!cancelled) setLoadingMemories(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [resolveErrorMessage, t]);

  const confirmDelete = React.useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      await deleteUserMemoryByID(token, deleteTarget.id);
      setItems((current) => current.filter((item) => item.id !== deleteTarget.id));
      setSelectedMemory((current) => current?.id === deleteTarget.id ? null : current);
      setDeleteTarget(null);
      toast.success(t("deleted"));
    } catch (error) {
      toast.error(t("deleteFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget, resolveErrorMessage, t]);

  return (
    <SettingsSection title={t("sectionTitle")}>
      <div className="flex flex-col gap-3">
        <SettingsFieldList>
          <SettingsFieldRow title={t("enabledTitle")} description={t("enabledDescription")}>
            <Switch
              checked={enabled}
              onCheckedChange={onEnabledChange}
              disabled={loading}
              aria-label={t("enabledTitle")}
            />
          </SettingsFieldRow>
        </SettingsFieldList>

        <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span>{t("listTitle")}</span>
          <span className="tabular-nums">
            {loadingMemories ? "--" : t("count", { count: items.length, limit: MEMORY_LIMIT })}
          </span>
        </div>

        {loadingMemories ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-14 w-full rounded-md" />
            <Skeleton className="h-14 w-4/5 rounded-md" />
          </div>
        ) : items.length === 0 ? (
          <p className="rounded-md bg-muted/30 px-3 py-3 text-xs text-muted-foreground">{t("empty")}</p>
        ) : (
          <div className="flex max-h-80 flex-col gap-1 overflow-y-auto pr-1">
            {items.map((item) => (
              <div key={item.id} className="group flex items-start gap-1 rounded-md hover:bg-muted/40">
                <button type="button" className="min-w-0 flex-1 px-3 py-2 text-left" onClick={() => setSelectedMemory(item)}>
                  <span className="line-clamp-2 break-words text-xs leading-5">{item.value}</span>
                </button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="mt-1 shrink-0 text-muted-foreground opacity-100 hover:text-destructive md:opacity-0 md:group-hover:opacity-100 md:group-focus-within:opacity-100"
                  onClick={() => setDeleteTarget(item)}
                  aria-label={t("delete")}
                >
                  <Trash2 />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      <Dialog open={selectedMemory !== null} onOpenChange={(open) => !open && setSelectedMemory(null)}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>{t("detailTitle")}</DialogTitle>
            <DialogDescription>
              {selectedMemory ? t("updatedAt", { time: new Date(selectedMemory.updatedAt).toLocaleString() }) : ""}
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-[50vh] overflow-y-auto whitespace-pre-wrap break-words text-sm leading-6">
            {selectedMemory?.value}
          </div>
          <DialogFooter>
            <Button variant="destructive" onClick={() => selectedMemory && setDeleteTarget(selectedMemory)}>
              <Trash2 data-icon="inline-start" />
              {t("delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteTarget !== null} onOpenChange={(open) => !open && !deleting && setDeleteTarget(null)}>
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>{t("confirmDeleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>{t("confirmDeleteDescription")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" disabled={deleting} onClick={() => void confirmDelete()}>
              {deleting ? t("deleting") : t("delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  );
}
