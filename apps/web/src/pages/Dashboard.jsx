import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  ArrowRight,
  Briefcase,
  Copy,
  DotsThree,
  FileArrowUp,
  FileText,
  GearSix,
  Plus,
  Sparkle,
  Trash,
  WarningCircle,
} from "@phosphor-icons/react";
import { api } from "../lib/api.js";
import { makeBlankResume, makeSampleResume } from "../data/sampleResume.js";
import { applyImportedResume } from "../lib/resume.js";
import { AISettingsDialog } from "../components/AISettingsDialog.jsx";
import { ImportResumeDialog } from "../components/ImportResumeDialog.jsx";
import { Button, Dialog, EmptyState, Field, IconButton, Spinner, Toast } from "../components/ui.jsx";

function formatUpdated(value) {
  if (!value) return "Just now";
  const date = new Date(value);
  const diff = Date.now() - date.getTime();
  if (diff < 60_000) return "Just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} hr ago`;
  return new Intl.DateTimeFormat("en", { month: "short", day: "numeric", year: date.getFullYear() === new Date().getFullYear() ? undefined : "numeric" }).format(date);
}

export function Dashboard() {
  const navigate = useNavigate();
  const [resumes, setResumes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ title: "Product Engineer CV", targetRole: "Product Engineer", starter: "sample" });
  const [creating, setCreating] = useState(false);
  const [menuId, setMenuId] = useState(null);
  const [toast, setToast] = useState("");
  const [providers, setProviders] = useState([]);
  const [aiSession, setAISession] = useState({ configured: false });
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsSaving, setSettingsSaving] = useState(false);
  const [settingsError, setSettingsError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [items, providerItems, session] = await Promise.all([api.listResumes(), api.providers(), api.getAISession()]);
      setResumes(Array.isArray(items) ? items : items?.items || []);
      setProviders(Array.isArray(providerItems) ? providerItems : providerItems?.providers || []);
      setAISession(session || { configured: false });
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function createResume(event) {
    event.preventDefault();
    setCreating(true);
    try {
      const payload = createForm.starter === "sample" ? makeSampleResume() : makeBlankResume(createForm.title);
      payload.title = createForm.title.trim() || "Untitled resume";
      if (createForm.targetRole.trim()) payload.document.basics.headline = createForm.targetRole.trim();
      const created = await api.createResume(payload);
      navigate(`/resume/${created.id}`);
    } catch (requestError) {
      setToast(requestError.message);
    } finally {
      setCreating(false);
    }
  }

  async function duplicate(id) {
    try {
      const created = await api.duplicateResume(id);
      setResumes((items) => [created, ...items]);
      setMenuId(null);
      setToast("Resume duplicated");
    } catch (requestError) { setToast(requestError.message); }
  }

  async function remove(id) {
    if (!window.confirm("Delete this resume? This cannot be undone.")) return;
    try {
      await api.deleteResume(id);
      setResumes((items) => items.filter((item) => item.id !== id));
      setMenuId(null);
      setToast("Resume deleted");
    } catch (requestError) { setToast(requestError.message); }
  }

  async function applyImport({ preview, linkedinURL }) {
    try {
      const payload = applyImportedResume(makeBlankResume(preview.candidate?.title || "Imported resume"), preview.candidate, { mode: "replace", linkedinURL });
      const created = await api.createResume(payload);
      setImportOpen(false);
      navigate(`/resume/${created.id}`);
    } catch (importError) { setToast(importError.message); }
  }

  async function saveAISettings(form) {
    setSettingsSaving(true);
    setSettingsError("");
    try {
      const session = await api.setAISession(form);
      setAISession(session);
      setSettingsOpen(false);
      setToast(`${providers.find((provider) => provider.id === form.provider)?.name || form.provider} connected for this session`);
    } catch (requestError) { setSettingsError(requestError.message); }
    finally { setSettingsSaving(false); }
  }

  async function clearAISettings() {
    setSettingsSaving(true);
    setSettingsError("");
    try {
      await api.clearAISession();
      setAISession({ configured: false });
      setSettingsOpen(false);
      setToast("AI provider disconnected");
    } catch (requestError) { setSettingsError(requestError.message); }
    finally { setSettingsSaving(false); }
  }

  return (
    <div className="dashboard-shell">
      <header className="dashboard-topbar">
        <a className="wordmark" href="/" aria-label="Forma home">FORMA</a>
        <div className="dashboard-topbar__actions">
          <button type="button" className="provider-status" onClick={() => setSettingsOpen(true)}><Sparkle size={17} />{aiSession.configured ? `${aiSession.provider} · ${aiSession.model}` : "Connect AI"}</button>
          <Button onClick={() => setCreateOpen(true)}><Plus size={17} />New resume</Button>
        </div>
      </header>

      <main className="dashboard-main">
        <section className="dashboard-hero">
          <div><span className="eyebrow">Your workspace</span><h1>Resumes built around facts,<br />not filler.</h1><p>Keep one master profile, tailor each version, and invite AI only when you want an editor.</p></div>
          <div className="dashboard-hero__note"><Briefcase size={21} /><div><strong>Private by default</strong><span>Editing and export stay local. Cloud AI runs only after an explicit click.</span></div></div>
        </section>

        <section className="resume-library" aria-labelledby="resume-library-title">
          <div className="resume-library__header">
            <div><h2 id="resume-library-title">Your resumes</h2><span>{resumes.length} {resumes.length === 1 ? "draft" : "drafts"}</span></div>
            <div className="resume-library__actions"><Button variant="ghost" onClick={() => setImportOpen(true)}><FileArrowUp size={17} />Import resume</Button><IconButton label="AI settings" onClick={() => setSettingsOpen(true)}><GearSix size={19} /></IconButton></div>
          </div>

          {loading ? <div className="library-loading"><Spinner label="Loading resumes" />Loading your workspace…</div> : error ? (
            <EmptyState icon={<WarningCircle size={28} />} title="The local API is unavailable" description={error} action={<Button onClick={load}>Try again</Button>} />
          ) : resumes.length === 0 ? (
            <EmptyState icon={<FileText size={30} />} title="Start with one honest draft" description="Add your facts first. Templates and AI edits come later." action={<Button onClick={() => setCreateOpen(true)}>Create a resume</Button>} />
          ) : (
            <div className="resume-list">
              <div className="resume-list__labels"><span>Resume</span><span>Template</span><span>Updated</span><span /></div>
              {resumes.map((resume) => (
                <article className="resume-list-row" key={resume.id} onDoubleClick={() => navigate(`/resume/${resume.id}`)}>
                  <button className="resume-list-row__main" type="button" onClick={() => navigate(`/resume/${resume.id}`)}>
                    <span className="resume-list-row__icon"><FileText size={21} /></span>
                    <span><strong>{resume.title}</strong><small>{resume.target_role || resume.document?.basics?.headline || "No target role"}</small></span>
                  </button>
                  <span className="resume-template-label">{resume.template || resume.document?.template || "Editorial"}</span>
                  <time dateTime={resume.updated_at}>{formatUpdated(resume.updated_at)}</time>
                  <div className="resume-list-row__actions"><button type="button" className="open-link" onClick={() => navigate(`/resume/${resume.id}`)}>Open <ArrowRight size={15} /></button><IconButton label={`Actions for ${resume.title}`} onClick={() => setMenuId(menuId === resume.id ? null : resume.id)}><DotsThree size={21} /></IconButton>{menuId === resume.id && <div className="row-menu"><button type="button" onClick={() => duplicate(resume.id)}><Copy size={16} />Duplicate</button><button type="button" className="danger" onClick={() => remove(resume.id)}><Trash size={16} />Delete</button></div>}</div>
                </article>
              ))}
            </div>
          )}
        </section>
      </main>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} title="Create a resume" description="Start from a realistic example or a clean document.">
        <form className="create-form" onSubmit={createResume}>
          <Field label="Document name"><input autoFocus required value={createForm.title} onChange={(event) => setCreateForm({ ...createForm, title: event.target.value })} /></Field>
          <Field label="Target role" hint="You can change it later."><input value={createForm.targetRole} onChange={(event) => setCreateForm({ ...createForm, targetRole: event.target.value })} /></Field>
          <fieldset className="starter-picker"><legend>Starting point</legend><label><input type="radio" name="starter" checked={createForm.starter === "sample"} onChange={() => setCreateForm({ ...createForm, starter: "sample" })} /><span><strong>Example content</strong><small>Explore the full editor, then replace it with your facts.</small></span></label><label><input type="radio" name="starter" checked={createForm.starter === "blank"} onChange={() => setCreateForm({ ...createForm, starter: "blank" })} /><span><strong>Blank resume</strong><small>Start with only the essential fields.</small></span></label></fieldset>
          <div className="dialog__actions"><Button variant="ghost" onClick={() => setCreateOpen(false)}>Cancel</Button><Button type="submit" disabled={creating}>{creating ? <><Spinner /> Creating</> : "Create resume"}</Button></div>
        </form>
      </Dialog>

      <AISettingsDialog open={settingsOpen} providers={providers} current={aiSession} onClose={() => setSettingsOpen(false)} onSave={saveAISettings} onClear={clearAISettings} saving={settingsSaving} error={settingsError} />
      <ImportResumeDialog open={importOpen} allowMerge={false} onClose={() => setImportOpen(false)} onPreview={(file) => api.previewImport(file)} onApply={applyImport} />
      <Toast message={toast} onDismiss={() => setToast("")} />
    </div>
  );
}
