import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { NicknameForm, validateName } from "./NicknameForm";

describe("validateName", () => {
  it("rejects empty input", () => {
    expect(validateName("")).toMatch(/입력/);
    expect(validateName("   ")).toMatch(/입력/);
  });

  it("rejects > 20 chars", () => {
    expect(validateName("a".repeat(21))).toMatch(/20자/);
  });

  it("rejects forbidden characters", () => {
    expect(validateName("hello!world")).toMatch(/허용되지/);
  });

  it("accepts Korean and ASCII names", () => {
    expect(validateName("철수")).toBeUndefined();
    expect(validateName("nick_name")).toBeUndefined();
    expect(validateName("Player 1")).toBeUndefined();
  });
});

describe("NicknameForm", () => {
  it("blocks submit on invalid input", () => {
    const onSubmit = vi.fn();
    render(<NicknameForm prompt="닉네임" onSubmit={onSubmit} />);
    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "입장" }));
    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("submits trimmed name on valid input", () => {
    const onSubmit = vi.fn();
    render(<NicknameForm prompt="닉네임" onSubmit={onSubmit} />);
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "  철수 " } });
    fireEvent.click(screen.getByRole("button", { name: "입장" }));
    expect(onSubmit).toHaveBeenCalledWith("철수");
  });
});
