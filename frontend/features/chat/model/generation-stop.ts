type CancelGeneration = (
  accessToken: string,
  runID: string,
) => Promise<{ canceled: boolean }>;

export function resolveAbortedSubmissionDisposition({
  requestDispatched,
}: {
  requestDispatched: boolean;
}): "restore_draft" | "await_server_reconciliation" {
  return requestDispatched ? "await_server_reconciliation" : "restore_draft";
}

export async function cancelGenerationAndReload({
  accessToken,
  runID,
  resolveAccessToken,
  cancelGeneration,
  reload,
}: {
  accessToken: string | null;
  runID: string;
  resolveAccessToken: () => Promise<string | null>;
  cancelGeneration: CancelGeneration;
  reload: () => void;
}): Promise<boolean> {
  try {
    const token = accessToken ?? (await resolveAccessToken());
    if (!token) {
      return false;
    }
    const result = await cancelGeneration(token, runID);
    return Boolean(result.canceled);
  } catch {
    return false;
  } finally {
    reload();
  }
}
