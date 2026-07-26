import { useEffect, useMemo, useState } from "react";
import { CheckCircle, Key, LockKey, WarningCircle } from "@phosphor-icons/react";
import { Button, Dialog, Field, Spinner } from "./ui.jsx";

export function AISettingsDialog({ open, providers, current, onClose, onSave, onClear, saving, error }) {
  const [form, setForm] = useState({ provider: "openai", model: "", api_key: "", base_url: "" });
  const selected = useMemo(() => providers.find((provider) => provider.id === form.provider) || providers[0], [providers, form.provider]);

  useEffect(() => {
    if (!open) return;
    const provider = current?.provider || providers[0]?.id || "openai";
    const preset = providers.find((item) => item.id === provider);
    setForm({
      provider,
      model: current?.model || preset?.suggested_model || "",
      api_key: "",
      base_url: current?.base_url || preset?.base_url || "",
    });
  }, [open, current, providers]);

  function chooseProvider(providerId) {
    const preset = providers.find((provider) => provider.id === providerId);
    setForm((value) => ({ ...value, provider: providerId, model: preset?.suggested_model || "", base_url: preset?.base_url || "", api_key: "" }));
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Connect an AI provider"
      description="The key is kept by the local Go server for this browser session only. It is never stored in PostgreSQL."
      className="dialog--settings"
    >
      <form className="settings-form" onSubmit={(event) => { event.preventDefault(); onSave(form); }}>
        <div className="provider-grid" role="radiogroup" aria-label="AI provider">
          {providers.map((provider) => (
            <button
              type="button"
              role="radio"
              aria-checked={form.provider === provider.id}
              className={`provider-option ${form.provider === provider.id ? "is-selected" : ""}`}
              key={provider.id}
              onClick={() => chooseProvider(provider.id)}
            >
              <span>{provider.name}</span>
              <small>{provider.local ? "Runs locally" : provider.protocol}</small>
              {form.provider === provider.id && <CheckCircle weight="fill" size={18} />}
            </button>
          ))}
        </div>

        <div className="settings-form__fields">
          <Field label="Model" hint="Model IDs change often, so this field always accepts a custom value.">
            <input required value={form.model} onChange={(event) => setForm({ ...form, model: event.target.value })} placeholder={selected?.suggested_model || "Provider model ID"} autoComplete="off" />
          </Field>
          <Field label="Base URL" hint={selected?.base_url_editable ? "Use the endpoint for your account region or workspace." : "Provider endpoint"}>
            <input required value={form.base_url} onChange={(event) => setForm({ ...form, base_url: event.target.value })} placeholder="https://api.example.com/v1" autoComplete="url" />
          </Field>
          <Field label={selected?.key_required === false ? "API key (optional)" : "API key"} hint="Do not paste keys into issues, logs, or screenshots.">
            <div className="secret-input"><Key size={18} /><input type="password" required={selected?.key_required !== false && !current?.configured} value={form.api_key} onChange={(event) => setForm({ ...form, api_key: event.target.value })} placeholder={current?.configured ? "Leave empty to keep the current session key" : "Paste your key"} autoComplete="off" spellCheck="false" /></div>
          </Field>
        </div>

        <div className="privacy-note">
          <LockKey size={20} />
          <div><strong>Local UI does not always mean local inference.</strong><span>Cloud providers receive the resume text you choose to review. Contact details are removed before the request. Ollama stays local.</span></div>
        </div>
        {error && <div className="form-error" role="alert"><WarningCircle size={18} />{error}</div>}
        <div className="dialog__actions">
          {current?.configured && onClear && <Button className="settings-disconnect" variant="ghost" onClick={onClear} disabled={saving}>Disconnect provider</Button>}
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={saving}>{saving ? <><Spinner label="Saving provider" /> Saving</> : "Use this provider"}</Button>
        </div>
      </form>
    </Dialog>
  );
}
