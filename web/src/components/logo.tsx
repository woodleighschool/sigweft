import { cn } from "@lib/utils";

interface LogoProps {
  className?: string;
}

export function Logo({ className }: LogoProps) {
  return (
    <svg
      fill="none"
      stroke="currentColor"
      strokeWidth={64}
      strokeLinecap="round"
      strokeLinejoin="round"
      viewBox="0 0 1024 1024"
      aria-hidden="true"
      className={cn("size-9 shrink-0", className)}
    >
      <path d="M192 608 C256 480 352 288 448 288 C544 288 480 512 416 608 C352 704 448 800 544 640 C592 560 640 544 672 608 C704 656 768 656 832 576" />
      <path d="M192 736 H312 M595 736 H832" />
    </svg>
  );
}
