import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AISettingsDialog } from "./AISettingsDialog.jsx";

describe("AISettingsDialog", () => {
  it("lets the user explicitly disconnect an ephemeral provider session", async () => {
    const user = userEvent.setup();
    const onClear = vi.fn();
    render(
      <AISettingsDialog
        open
        providers={[{ id: "ollama", name: "Ollama", protocol: "ollama_chat", suggested_model: "qwen3", base_url: "http://localhost:11434/v1", key_required: false, local: true }]}
        current={{ configured: true, provider: "ollama", model: "qwen3" }}
        onClose={vi.fn()}
        onSave={vi.fn()}
        onClear={onClear}
        saving={false}
        error=""
      />,
    );

    await user.click(screen.getByRole("button", { name: "Disconnect provider" }));
    expect(onClear).toHaveBeenCalledTimes(1);
  });
});
