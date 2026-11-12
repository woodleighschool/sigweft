import { useEffect, useRef, useState } from "react";
import { CheckIcon, CopyIcon } from "lucide-react";
import { Button } from "@components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@components/ui/tooltip";
import type { Result } from "@lib/api-client/types.gen";
import { clipboardContent } from "@lib/form";
import { SetupInstructions } from "@components/setup-instructions";

export function Preview({
  result,
  ready,
  copyHint,
  pending,
  inputKey,
  clients,
}: {
  result?: Result;
  ready: boolean;
  copyHint?: string;
  pending: boolean;
  inputKey: string;
  clients: string[];
}) {
  const frame = useRef<HTMLIFrameElement>(null);
  const [dimensions, setDimensions] = useState({ width: 0, height: 320 });
  const [notice, setNotice] = useState({ key: "", text: "", copied: false });
  const copyResetTimer = useRef<number | null>(null);
  useEffect(
    () => () => {
      if (copyResetTimer.current !== null) window.clearTimeout(copyResetTimer.current);
    },
    [],
  );
  const currentNotice = notice.key === inputKey ? notice : null;
  const document = result
    ? `<!doctype html><html><head><meta charset="utf-8"><style>html{color-scheme:only light;background:#fff;color:#000}body{margin:0;width:max-content;min-width:0}</style></head><body>${result.html}</body></html>`
    : "";

  async function copy() {
    if (!ready || !result) return;
    if (copyResetTimer.current !== null) window.clearTimeout(copyResetTimer.current);
    try {
      await navigator.clipboard.write([new ClipboardItem(clipboardContent(result))]);
      setNotice({
        key: inputKey,
        text: "",
        copied: true,
      });
      copyResetTimer.current = window.setTimeout(
        () => setNotice({ key: "", text: "", copied: false }),
        2000,
      );
    } catch {
      const content = frame.current?.contentDocument;
      if (content) {
        const range = content.createRange();
        range.selectNodeContents(content.body);
        const selection = content.getSelection();
        selection?.removeAllRanges();
        selection?.addRange(range);
        frame.current?.contentWindow?.focus();
      }
      setNotice({
        key: inputKey,
        text: content
          ? "Automatic copying was blocked. The signature is selected; press Command-C or Ctrl-C."
          : "Could not copy the signature. Try again.",
        copied: false,
      });
    }
  }

  return (
    <div className="flex min-w-0 flex-col gap-8 lg:sticky lg:top-8 lg:self-start">
      <section aria-labelledby="preview-title" className="flex min-w-0 flex-col gap-5">
        <div className="flex min-h-8 items-center gap-3">
          <h2 id="preview-title" className="text-lg font-semibold">
            2. Preview
          </h2>
          <TooltipProvider>
            <Tooltip disabled={!copyHint}>
              <TooltipTrigger
                render={
                  <Button
                    disabled={!ready}
                    focusableWhenDisabled
                    onClick={() => void copy()}
                    aria-live="polite"
                  />
                }
              >
                {currentNotice?.copied ? (
                  <CheckIcon data-icon="inline-start" />
                ) : (
                  <CopyIcon data-icon="inline-start" />
                )}
                {currentNotice?.copied ? "Copied" : "Copy"}
              </TooltipTrigger>
              <TooltipContent>{copyHint}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
        <div
          className="overflow-x-auto rounded-xl border bg-white p-5 sm:p-7"
          aria-label="Signature preview"
          aria-busy={pending}
        >
          {result ? (
            <iframe
              ref={frame}
              title="Signature preview"
              sandbox="allow-same-origin"
              srcDoc={document}
              className="block w-full border-0"
              style={{ minWidth: dimensions.width, height: dimensions.height }}
              onLoad={() => {
                const body = frame.current?.contentDocument?.body;
                if (body)
                  setDimensions({
                    width: Math.ceil(body.getBoundingClientRect().width),
                    height: Math.ceil(body.getBoundingClientRect().height) + 2,
                  });
              }}
            />
          ) : (
            <div className="h-56" />
          )}
        </div>
        {currentNotice?.text && (
          <p role="alert" className="text-sm text-destructive">
            {currentNotice.text}
          </p>
        )}
      </section>
      <SetupInstructions clients={clients} />
    </div>
  );
}
