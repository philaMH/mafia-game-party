import { useEffect } from "react";

interface Toast {
  code: string;
  message: string;
  addedAt: number;
}

interface Props {
  errors: Toast[];
  onDismiss: (addedAt: number) => void;
}

const TTL_MS = 5000;

// ToastList auto-dismisses each entry 5 s after it was added (BR-U5-ERR-2).
// We schedule a single timer for the oldest entry; expired older entries
// are cleared on each render so the list never piles up.
export function ToastList({ errors, onDismiss }: Props) {
  useEffect(() => {
    if (errors.length === 0) return;
    const oldest = errors[0];
    if (!oldest) return;
    const remaining = Math.max(0, TTL_MS - (Date.now() - oldest.addedAt));
    const timer = setTimeout(() => onDismiss(oldest.addedAt), remaining);
    return () => clearTimeout(timer);
  }, [errors, onDismiss]);

  if (errors.length === 0) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        position: "fixed",
        top: "1rem",
        right: "1rem",
        display: "flex",
        flexDirection: "column",
        gap: "0.5rem",
        maxWidth: "20rem",
      }}
    >
      {errors.map((e) => (
        <div
          key={e.addedAt}
          onClick={() => onDismiss(e.addedAt)}
          style={{
            background: "var(--card)",
            color: "var(--warn)",
            border: "1px solid var(--warn)",
            padding: "0.5rem 0.75rem",
            borderRadius: "0.375rem",
            cursor: "pointer",
          }}
        >
          {e.message}
        </div>
      ))}
    </div>
  );
}
