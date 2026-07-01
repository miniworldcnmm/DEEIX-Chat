"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ChatContentWidth } from "@/shared/model/chat-content-width";
import { useAppearancePreferencesPersistence } from "@/features/settings/hooks/use-appearance-preferences-persistence";
import { useSettingsChat } from "@/features/settings/hooks/use-settings-chat";
import {
  type ChatFontOption,
  type ChatFontWeightOption,
  useChatFontPreference,
  useChatFontWeightPreference,
  writeChatFontPreference,
  writeChatFontWeightPreference,
} from "@/features/settings/utils/chat-font";
import { ModelSelect, type ModelSelectOption } from "@/shared/components/model-select";
import {
  SettingsFieldList,
  SettingsFieldRow,
  SettingsPage,
  SettingsSection,
  SettingsSectionSeparator,
} from "@/shared/components/settings-layout";
import { resolveModelOptionIconUrl, resolveModelOptionLabel } from "@/shared/lib/model-option-display";
import { parseKindsJSON } from "@/shared/model/llm-schema";
import { platformModifierLabel, platformSendShortcut } from "@/shared/lib/platform-shortcuts";
import type { SendShortcut } from "@/features/settings/types/settings";
import { ChatDisplayAppearance } from "./chat-display-appearance";
import { MemorySettingsSection } from "./memory-settings-section";

type ModelOption = ModelSelectOption;

const SYSTEM_RECOMMENDED_MODEL = "none";

const DEFAULT_REASONING_EFFORT_OPTIONS = ["default", "low", "medium", "high", "xhigh", "max"] as const;
const CUSTOM_REASONING_EFFORT_VALUE = "__custom__";

function ReasoningEffortSelect({
  value,
  onChange,
  disabled,
}: {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  const t = useTranslations("settings.chatPage.defaultModelParams");
  const isCustom = value !== "" && !(DEFAULT_REASONING_EFFORT_OPTIONS as readonly string[]).includes(value);
  const selectValue = isCustom ? CUSTOM_REASONING_EFFORT_VALUE : value === "" ? "default" : value;
  const [customDraft, setCustomDraft] = React.useState(value);

  React.useEffect(() => {
    setCustomDraft(value);
  }, [value]);

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <Select
        value={selectValue}
        onValueChange={(next) => {
          if (next === CUSTOM_REASONING_EFFORT_VALUE) {
            onChange(customDraft.trim() === "" ? "" : customDraft.trim());
            return;
          }
          onChange(next);
        }}
        disabled={disabled}
      >
        <SelectTrigger size="sm" className="w-40 text-left">
          <SelectValue />
        </SelectTrigger>
        <SelectContent align="start">
          <SelectItem value="default">{t("effortDefault")}</SelectItem>
          {DEFAULT_REASONING_EFFORT_OPTIONS.filter((v) => v !== "default").map((v) => (
            <SelectItem key={v} value={v}>
              {t(`effort_${v}`)}
            </SelectItem>
          ))}
          <SelectItem value={CUSTOM_REASONING_EFFORT_VALUE}>{t("effortCustom")}</SelectItem>
        </SelectContent>
      </Select>
      {isCustom ? (
        <Input
          value={customDraft}
          onChange={(event) => {
            setCustomDraft(event.target.value);
            onChange(event.target.value.trim());
          }}
          disabled={disabled}
          placeholder={t("effortCustomPlaceholder")}
          className="w-40"
        />
      ) : null}
    </div>
  );
}

// Main component.

export function SettingsChat() {
  const t = useTranslations("settings.chatPage");
  const {
    settings,
    loading,
    billingMode,
    contextCompressionEnabled,
    vendorGroups,
    handleBool,
    handleEnum,
    handleDefaultModel,
    handleDefaultTemperature,
    handleDefaultReasoningEffort,
  } = useSettingsChat();
  const billingEnabled = billingMode !== "self";
  const chatFont = useChatFontPreference();
  const chatFontWeight = useChatFontWeightPreference();
  const persistAppearancePreferences = useAppearancePreferencesPersistence();
  const [modifierLabel, setModifierLabel] = React.useState<"Command" | "Ctrl">("Ctrl");
  const [modifierShortcut, setModifierShortcut] = React.useState<Exclude<SendShortcut, "enter">>("ctrl_enter");
  const modelOptions = React.useMemo<ModelOption[]>(
    () => [
      { label: t("defaultModel.systemRecommended"), value: SYSTEM_RECOMMENDED_MODEL, iconUrl: null },
      ...vendorGroups.flatMap(([, items]) =>
        items
          .filter((model) => model.platformModelName.trim() && parseKindsJSON(model.kindsJSON).includes("chat"))
          .map((model) => ({
            label: resolveModelOptionLabel(model.platformModelName),
            value: model.platformModelName,
            iconUrl: resolveModelOptionIconUrl({
              platformModelName: model.platformModelName,
              vendor: model.vendor ?? "",
              icon: model.icon ?? "",
            }),
          })),
      ),
    ],
    [t, vendorGroups],
  );

  React.useEffect(() => {
    setModifierLabel(platformModifierLabel());
    setModifierShortcut(platformSendShortcut());
  }, []);

  const sendShortcutLabel = settings.sendShortcut === "enter" ? "Enter" : `${modifierLabel}+Enter`;

  const handleChatFontChange = React.useCallback((value: ChatFontOption) => {
    writeChatFontPreference(value);
    persistAppearancePreferences({ chatFont: value });
  }, [persistAppearancePreferences]);

  const handleChatFontWeightChange = React.useCallback((value: ChatFontWeightOption) => {
    writeChatFontWeightPreference(value);
    persistAppearancePreferences({ chatFontWeight: value });
  }, [persistAppearancePreferences]);

  const handleContentWidthChange = React.useCallback((value: ChatContentWidth) => {
    handleEnum("chat.content_width", "contentWidth")(value);
  }, [handleEnum]);

  return (
    <SettingsPage>
      <SettingsSection title={t("defaultModel.sectionTitle")}>
        <SettingsFieldList>
          <SettingsFieldRow
            title={t("defaultModel.title")}
            description={t("defaultModel.description")}
          >
            {loading ? (
              <Skeleton className="h-8 w-full rounded-md" />
            ) : (
              <ModelSelect
                value={settings.defaultModel}
                fallbackValue={SYSTEM_RECOMMENDED_MODEL}
                options={modelOptions}
                contentClassName="min-w-[min(320px,calc(100vw-2rem))]"
                onChange={handleDefaultModel}
                disabled={loading}
              />
            )}
          </SettingsFieldRow>
          <div className="pt-4">
            <SettingsFieldRow
              title={t("defaultModel.autoTitle")}
              description={t("defaultModel.autoTitleDescription")}
            >
              <Switch
                checked={settings.autoGenerateTitle}
                onCheckedChange={handleBool("chat.auto_generate_title", "autoGenerateTitle")}
                disabled={loading}
                aria-label={t("defaultModel.autoTitle")}
              />
            </SettingsFieldRow>
          </div>
        </SettingsFieldList>
      </SettingsSection>

      <SettingsSectionSeparator />

      <SettingsSection title={t("defaultModelParams.sectionTitle")}>
        <SettingsFieldList>
          <SettingsFieldRow
            title={t("defaultModelParams.thinkingEnabledTitle")}
            description={t("defaultModelParams.thinkingEnabledDescription")}
          >
            <Switch
              checked={settings.defaultThinkingEnabled}
              onCheckedChange={handleBool("chat.default_thinking_enabled", "defaultThinkingEnabled")}
              disabled={loading}
              aria-label={t("defaultModelParams.thinkingEnabledTitle")}
            />
          </SettingsFieldRow>
          <SettingsFieldRow
            title={t("defaultModelParams.temperatureTitle")}
            description={t("defaultModelParams.temperatureDescription")}
          >
            <Input
              type="number"
              min={0}
              max={2}
              step={0.1}
              value={settings.defaultTemperature}
              onChange={(event) => {
                const parsed = Number(event.target.value);
                if (Number.isFinite(parsed) && parsed >= 0 && parsed <= 2) {
                  handleDefaultTemperature(parsed);
                }
              }}
              disabled={loading}
              className="w-24"
            />
          </SettingsFieldRow>
          <SettingsFieldRow
            title={t("defaultModelParams.reasoningEffortTitle")}
            description={t("defaultModelParams.reasoningEffortDescription")}
          >
            <ReasoningEffortSelect
              value={settings.defaultReasoningEffort}
              onChange={handleDefaultReasoningEffort}
              disabled={loading}
            />
          </SettingsFieldRow>
        </SettingsFieldList>
      </SettingsSection>

      <SettingsSectionSeparator />

      <SettingsSection title={t("input.sectionTitle")}>
        <SettingsFieldList>
          <SettingsFieldRow
            title={t("input.shortcutTitle")}
            description={t("input.shortcutDescription", { shortcut: sendShortcutLabel })}
          >
            <Select
              value={settings.sendShortcut === "enter" ? "enter" : modifierShortcut}
              onValueChange={handleEnum("chat.send_on_enter", "sendShortcut")}
              disabled={loading}
            >
              <SelectTrigger size="sm" className="text-left md:text-right *:data-[slot=select-value]:flex-1 *:data-[slot=select-value]:justify-start md:*:data-[slot=select-value]:justify-end">
                <SelectValue />
              </SelectTrigger>
              <SelectContent align="start">
                <SelectItem value="enter">Enter</SelectItem>
                <SelectItem value={modifierShortcut}>{modifierLabel}+Enter</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldRow>
          <div className="pt-4">
            <SettingsFieldRow
              title={t("input.heightTitle")}
              description={t("input.heightDescription")}
            >
              <Select
                value={settings.inputHeight}
                onValueChange={handleEnum("chat.input_height", "inputHeight")}
                disabled={loading}
              >
                <SelectTrigger size="sm" className="text-left md:text-right *:data-[slot=select-value]:flex-1 *:data-[slot=select-value]:justify-start md:*:data-[slot=select-value]:justify-end">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  <SelectItem value="compact">{t("input.height.compact")}</SelectItem>
                  <SelectItem value="standard">{t("input.height.standard")}</SelectItem>
                  <SelectItem value="loose">{t("input.height.loose")}</SelectItem>
                </SelectContent>
              </Select>
            </SettingsFieldRow>
          </div>
          <div className="pt-4">
            <SettingsFieldRow
              title={t("input.restoreDraftTitle")}
              description={t("input.restoreDraftDescription")}
            >
              <Switch
                checked={settings.restoreDraftOnFailure}
                onCheckedChange={handleBool("chat.restore_draft_on_failure", "restoreDraftOnFailure")}
                disabled={loading}
                aria-label={t("input.restoreDraftTitle")}
              />
            </SettingsFieldRow>
          </div>
          <div className="pt-4">
            <SettingsFieldRow
              title={t("input.preserveDraftTitle")}
              description={t("input.preserveDraftDescription")}
            >
              <Switch
                checked={settings.preserveConversationDrafts}
                onCheckedChange={handleBool("chat.preserve_conversation_drafts", "preserveConversationDrafts")}
                disabled={loading}
                aria-label={t("input.preserveDraftTitle")}
              />
            </SettingsFieldRow>
          </div>
          <div className="pt-4">
            <SettingsFieldRow
              title={t("input.reuseModelOptionsTitle")}
              description={t("input.reuseModelOptionsDescription")}
            >
              <Switch
                checked={settings.reuseModelOptions}
                onCheckedChange={handleBool("chat.reuse_model_options", "reuseModelOptions")}
                disabled={loading}
                aria-label={t("input.reuseModelOptionsTitle")}
              />
            </SettingsFieldRow>
          </div>
          <div className="pt-4">
            <SettingsFieldRow
              title={t("input.deleteFilesDefaultTitle")}
              description={t("input.deleteFilesDefaultDescription")}
            >
              <Switch
                checked={settings.deleteFilesByDefault}
                onCheckedChange={handleBool("chat.delete_conversation_files_by_default", "deleteFilesByDefault")}
                disabled={loading}
                aria-label={t("input.deleteFilesDefaultTitle")}
              />
            </SettingsFieldRow>
          </div>
        </SettingsFieldList>
      </SettingsSection>

      <SettingsSectionSeparator />

      <SettingsSection title={t("display.sectionTitle")}>
        <SettingsFieldList>
          <div>
            <SettingsFieldRow
              title={t("display.markdownTitle")}
              description={t("display.markdownDescription")}
            >
              <Switch
                checked={settings.markdownRender}
                onCheckedChange={handleBool("chat.markdown_render", "markdownRender")}
                disabled={loading}
                aria-label={t("display.markdownTitle")}
              />
            </SettingsFieldRow>
          </div>

          <div className="pt-4">
            <SettingsFieldRow
              title={t("display.modelTitle")}
              description={t("display.modelDescription")}
            >
              <Switch
                checked={settings.showModelInfo}
                onCheckedChange={handleBool("chat.show_model_info", "showModelInfo")}
                disabled={loading}
                aria-label={t("display.modelTitle")}
              />
            </SettingsFieldRow>
          </div>

          <div className="pt-4">
            <SettingsFieldRow
              title={t("display.tokenTitle")}
              description={t("display.tokenDescription")}
            >
              <Switch
                checked={settings.showTokenUsage}
                onCheckedChange={handleBool("chat.show_token_usage", "showTokenUsage")}
                disabled={loading}
                aria-label={t("display.tokenTitle")}
              />
            </SettingsFieldRow>
          </div>

          <div className="pt-4">
            <SettingsFieldRow
              title={t("display.latencyTitle")}
              description={t("display.latencyDescription")}
            >
              <Switch
                checked={settings.showLatency}
                onCheckedChange={handleBool("chat.show_latency", "showLatency")}
                disabled={loading}
                aria-label={t("display.latencyTitle")}
              />
            </SettingsFieldRow>
          </div>

          <div className="pt-4">
            <SettingsFieldRow
              title={t("display.costTitle")}
              description={billingEnabled ? t("display.costDescription") : t("display.costDescriptionSelfMode")}
            >
              <Switch
                checked={billingEnabled && settings.showBillingCost}
                onCheckedChange={handleBool("chat.show_billing_cost", "showBillingCost")}
                disabled={loading || !billingEnabled}
                aria-label={t("display.costTitle")}
              />
            </SettingsFieldRow>
          </div>

          <div className="pt-4">
            <ChatDisplayAppearance
              contentWidth={settings.contentWidth}
              chatFont={chatFont}
              chatFontWeight={chatFontWeight}
              onContentWidthChange={handleContentWidthChange}
              onChatFontChange={handleChatFontChange}
              onChatFontWeightChange={handleChatFontWeightChange}
              disabled={loading}
            />
          </div>
        </SettingsFieldList>
      </SettingsSection>

      <SettingsSectionSeparator />

      {contextCompressionEnabled ? (
        <>
          <SettingsSection title={t("context.sectionTitle")}>
            <SettingsFieldList>
              <SettingsFieldRow
                title={t("context.autoCompactTitle")}
                description={t("context.autoCompactDescription")}
              >
                <Switch
                  checked={settings.contextCompactAuto}
                  onCheckedChange={handleBool("chat.context_compact_auto", "contextCompactAuto")}
                  disabled={loading}
                  aria-label={t("context.autoCompactTitle")}
                />
              </SettingsFieldRow>
            </SettingsFieldList>
          </SettingsSection>

          <SettingsSectionSeparator />
        </>
      ) : null}

      <SettingsSection title={t("file.sectionTitle")}>
        <SettingsFieldList>
          <SettingsFieldRow
            title={t("file.modeTitle")}
            description={
              settings.fileMode === "auto"
                ? t("file.modeDescription.auto")
                : settings.fileMode === "full_context"
                  ? t("file.modeDescription.fullContext")
                  : t("file.modeDescription.rag")
            }
          >
            <Select
              value={settings.fileMode}
              onValueChange={handleEnum("chat.file_mode", "fileMode")}
              disabled={loading}
            >
              <SelectTrigger size="sm" className="text-left md:text-right *:data-[slot=select-value]:flex-1 *:data-[slot=select-value]:justify-start md:*:data-[slot=select-value]:justify-end">
                <SelectValue />
              </SelectTrigger>
              <SelectContent align="start">
                <SelectItem value="auto">{t("file.mode.auto")}</SelectItem>
                <SelectItem value="full_context">{t("file.mode.fullContext")}</SelectItem>
                <SelectItem value="rag">{t("file.mode.rag")}</SelectItem>
              </SelectContent>
            </Select>
          </SettingsFieldRow>
        </SettingsFieldList>
      </SettingsSection>

      <SettingsSectionSeparator />

      <MemorySettingsSection
        enabled={settings.memoryEnabled}
        loading={loading}
        onEnabledChange={handleBool("chat.memory_enabled", "memoryEnabled")}
      />
    </SettingsPage>
  );
}
