import { authedRequest } from "@/shared/api/authed-client";

export type UserSettingsMap = Record<string, string>;

type UserSettingsResponse = {
  settings: UserSettingsMap;
};

export type UserModelOptionPayload = {
  thinkingEnabled?: boolean;
  temperature?: number;
  reasoningEffort?: string;
};

export type UserModelOptionsMap = Record<string, UserModelOptionPayload>;

type ListUserModelOptionsResponse = {
  options: UserModelOptionsMap;
};

export async function getUserSettings(accessToken: string): Promise<UserSettingsMap> {
  const data = await authedRequest<UserSettingsResponse>("/api/v1/user/settings", { accessToken }, true);
  return data.settings ?? {};
}

export async function patchUserSettings(
  accessToken: string,
  settings: UserSettingsMap,
): Promise<UserSettingsMap> {
  const data = await authedRequest<UserSettingsResponse>(
    "/api/v1/user/settings",
    {
      accessToken,
      method: "PATCH",
      body: JSON.stringify({ settings }),
    },
    true,
  );
  return data.settings ?? {};
}

export async function listUserModelOptions(accessToken: string): Promise<UserModelOptionsMap> {
  const data = await authedRequest<ListUserModelOptionsResponse>(
    "/api/v1/user/settings/model-options",
    { accessToken },
    true,
  );
  return data.options ?? {};
}

export async function upsertUserModelOption(
  accessToken: string,
  platformModelName: string,
  payload: UserModelOptionPayload,
): Promise<UserModelOptionsMap> {
  const encoded = encodeURIComponent(platformModelName.trim());
  const data = await authedRequest<ListUserModelOptionsResponse>(
    `/api/v1/user/settings/model-options/${encoded}`,
    {
      accessToken,
      method: "PUT",
      body: JSON.stringify(payload),
    },
    true,
  );
  return data.options ?? {};
}

export async function deleteUserModelOption(
  accessToken: string,
  platformModelName: string,
): Promise<UserModelOptionsMap> {
  const encoded = encodeURIComponent(platformModelName.trim());
  const data = await authedRequest<ListUserModelOptionsResponse>(
    `/api/v1/user/settings/model-options/${encoded}`,
    {
      accessToken,
      method: "DELETE",
    },
    true,
  );
  return data.options ?? {};
}
