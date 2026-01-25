import { useCallback } from "react";
import type { StreamEvent } from "@/types";

interface UseSSEStreamOptions {
  onEvent: (event: StreamEvent) => void;
  onError: (error: string) => void;
  onComplete: () => void;
}

export function useSSEStream({ onEvent, onError, onComplete }: UseSSEStreamOptions) {

  const processStream = useCallback(
    async (response: Response) => {
      if (!response.ok) {
        const data = await response.json();
        onError(data.error || "Request failed");
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        onError("Failed to get response reader");
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          for (const line of lines) {
            if (line.startsWith("data: ")) {
              const data = line.slice(6);
              if (data) {
                try {
                  const event: StreamEvent = JSON.parse(data);
                  onEvent(event);
                } catch {
                  // Parse error - skip this line
                }
              }
            }
          }
        }
        onComplete();
      } catch (error) {
        // AbortError は正常な中断操作なので、エラーとして扱わない
        if (error instanceof Error && error.name === "AbortError") {
          onComplete();
          return;
        }
        onError(error instanceof Error ? error.message : "Stream error");
        onComplete();
      }
    },
    [onEvent, onError, onComplete]
  );

  return { processStream };
}
