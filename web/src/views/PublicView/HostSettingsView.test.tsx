import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { GameContext, type GameContextValue } from "../../context/GameContext";
import { initialState } from "../../context/reducer";
import { defaultOptions } from "../../types/wire";

import { HostSettingsView } from "./HostSettingsView";

function makeCtx(overrides: Partial<GameContextValue> = {}): GameContextValue {
  return {
    ...initialState,
    send: vi.fn(),
    toggleVoice: vi.fn(),
    ackError: vi.fn(),
    logout: vi.fn(),
    hostOptions: defaultOptions(8),
    saveHostOptions: vi.fn(),
    hostToken: "tok-host", // host-claim has happened
    ...overrides,
  };
}

function renderSettings(ctx: GameContextValue) {
  return render(
    <GameContext.Provider value={ctx}>
      <MemoryRouter initialEntries={["/public/settings"]}>
        <Routes>
          <Route
            path="/public"
            element={<div data-testid="public-route">PUBLIC</div>}
          />
          <Route path="/public/settings" element={<HostSettingsView />} />
        </Routes>
      </MemoryRouter>
    </GameContext.Provider>,
  );
}

describe("HostSettingsView", () => {
  it("clicking 저장 후 메인으로 calls saveHostOptions and navigates back", () => {
    const saveHostOptions = vi.fn();
    const start = { ...defaultOptions(8), discussionSeconds: 60 };
    renderSettings(makeCtx({ saveHostOptions, hostOptions: start }));

    // Change a numeric field and a checkbox.
    const discussion = screen.getByDisplayValue("60");
    fireEvent.change(discussion, { target: { value: "240" } });

    const selfHeal = screen.getByLabelText("의사 자가치료 허용") as HTMLInputElement;
    fireEvent.click(selfHeal);

    fireEvent.click(screen.getByRole("button", { name: /저장 후 메인으로/ }));

    expect(saveHostOptions).toHaveBeenCalledTimes(1);
    const call = saveHostOptions.mock.calls[0];
    if (!call) throw new Error("saveHostOptions was not called");
    const saved = call[0];
    expect(saved.discussionSeconds).toBe(240);
    expect(saved.doctorSelfHealAllowed).toBe(false); // toggled from default true
    expect(screen.getByTestId("public-route")).toBeInTheDocument();
  });

  it("shows the recommended-config warning when mafiaCount is far from default", () => {
    const start = { ...defaultOptions(8), mafiaCount: 5 }; // default(8).mafiaCount === 2; |5-2|=3 > 1
    renderSettings(makeCtx({ hostOptions: start }));
    expect(screen.getByText(/권장하지 않는 설정/)).toBeInTheDocument();
    // Saving must still be allowed (Q7=A — warning only).
    expect(screen.getByRole("button", { name: /저장 후 메인으로/ })).not.toBeDisabled();
  });

  it("redirects non-host visitors to /public", () => {
    renderSettings(makeCtx({ hostToken: undefined }));
    expect(screen.getByTestId("public-route")).toBeInTheDocument();
  });

  it("redirects when room is already opened", () => {
    renderSettings(makeCtx({ roomOpened: true }));
    expect(screen.getByTestId("public-route")).toBeInTheDocument();
  });
});
