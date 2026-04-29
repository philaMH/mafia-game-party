import { useState, type FormEvent } from "react";

const NAME_RE = /^[가-힣a-zA-Z0-9 _-]+$/;

export function validateName(raw: string): string | undefined {
  const t = raw.trim();
  if (t.length === 0) return "닉네임을 입력하세요";
  if (t.length > 20) return "닉네임은 20자 이하입니다";
  if (!NAME_RE.test(t)) return "허용되지 않는 문자가 있습니다";
  return undefined;
}

interface Props {
  prompt: string;
  onSubmit: (name: string) => void;
}

export function NicknameForm({ prompt, onSubmit }: Props) {
  const [name, setName] = useState("");
  const [error, setError] = useState<string | undefined>();

  const submit = (ev: FormEvent) => {
    ev.preventDefault();
    const err = validateName(name);
    if (err) {
      setError(err);
      return;
    }
    setError(undefined);
    onSubmit(name.trim());
  };

  return (
    <form onSubmit={submit} aria-label={prompt}>
      <label
        style={{
          display: "block",
          fontSize: "1.125rem",
          marginBottom: "0.5rem",
          color: "var(--fg)",
        }}
      >
        {prompt}
      </label>
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        maxLength={20}
        autoFocus
        aria-invalid={!!error}
        aria-describedby={error ? "nickname-error" : undefined}
        style={{ width: "100%", maxWidth: "20rem" }}
      />
      <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.75rem" }}>
        <button type="submit">입장</button>
      </div>
      {error && (
        <p
          id="nickname-error"
          role="alert"
          style={{ color: "var(--warn)", marginTop: "0.5rem" }}
        >
          {error}
        </p>
      )}
    </form>
  );
}
