import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  CaretDown,
  CloudCheck,
  Copy,
  DotsThree,
  DownloadSimple,
  Eye,
  FileArrowUp,
  FileCode,
  FileDoc,
  FilePdf,
  Trash,
  WarningCircle,
} from "@phosphor-icons/react";
import { api } from "../lib/api.js";
import {
  addArrayItem,
  applyImportedResume,
  applyReviewSuggestionWithResult,
  exportResumeJSON,
  moveSection,
  toggleSection,
} from "../lib/resume.js";
import { sectionCatalog } from "../data/sampleResume.js";
import { AIReviewPanel } from "../components/AIReviewPanel.jsx";
import { AISettingsDialog } from "../components/AISettingsDialog.jsx";
import { ImportResumeDialog } from "../components/ImportResumeDialog.jsx";
import { ResumeDocument } from "../components/ResumeDocument.jsx";
import { SectionEditorPanel } from "../components/SectionEditorPanel.jsx";
import { SectionRail } from "../components/SectionRail.jsx";
import { TemplatesDialog } from "../components/TemplatesDialog.jsx";
import { Button, Dialog, EmptyState, IconButton, InlineEdit, Spinner, Toast } from "../components/ui.jsx";

const MIN_RULE_PROGRESS_MS = 500;
const MIN_VALIDATION_PROGRESS_MS = 350;

function keepPhaseVisible(startedAt, minimumMs) {
  const remaining = minimumMs - (Date.now() - startedAt);
  return remaining > 0 ? new Promise((resolve) => window.setTimeout(resolve, remaining)) : Promise.resolve();
}

export function Editor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const previewRef = useRef(null);
  const saveTimer = useRef(null);
  const exportMenuRef = useRef(null);
  const dirty = useRef(false);
  const [resume, setResume] = useState(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [saveStatus, setSaveStatus] = useState("saved");
  const [selectedSection, setSelectedSection] = useState("basics");
  const [panelView, setPanelView] = useState("section");
  const [previewMode, setPreviewMode] = useState(false);
  const [templatesOpen, setTemplatesOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);
  const [exportMenuOpen, setExportMenuOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const [toast, setToast] = useState("");
  const [providers, setProviders] = useState([]);
  const [aiSession, setAISession] = useState({ configured: false });
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsSaving, setSettingsSaving] = useState(false);
  const [settingsError, setSettingsError] = useState("");
  const [pendingReview, setPendingReview] = useState(false);
  const [reviewStatus, setReviewStatus] = useState("idle");
  const [reviewPhase, setReviewPhase] = useState("rules");
  const [reviewError, setReviewError] = useState("");
  const [review, setReview] = useState(null);
  const [quality, setQuality] = useState(null);
  const [exportingFormat, setExportingFormat] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const [loaded, providerItems, session] = await Promise.all([api.getResume(id), api.providers(), api.getAISession()]);
        if (cancelled) return;
        setResume(loaded);
        setProviders(Array.isArray(providerItems) ? providerItems : providerItems?.providers || []);
        setAISession(session || { configured: false });
      } catch (error) {
        if (!cancelled) setLoadError(error.message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [id]);

  const persist = useCallback(async (nextResume) => {
    setSaveStatus("saving");
    try {
      const saved = await api.updateResume(id, { title: nextResume.title, document: nextResume.document });
      setSaveStatus("saved");
      dirty.current = false;
      setResume((current) => current ? { ...current, updated_at: saved.updated_at || current.updated_at } : current);
    } catch (error) {
      setSaveStatus("error");
      setToast(error.message);
    }
  }, [id]);

  useEffect(() => {
    if (!resume || !dirty.current) return undefined;
    window.clearTimeout(saveTimer.current);
    saveTimer.current = window.setTimeout(() => persist(resume), 700);
    return () => window.clearTimeout(saveTimer.current);
  }, [resume, persist]);

  useEffect(() => {
    const warn = (event) => {
      if (!dirty.current) return;
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, []);

  useEffect(() => {
    if (!exportMenuOpen) return undefined;

    const closeOnOutsideClick = (event) => {
      if (!exportMenuRef.current?.contains(event.target)) setExportMenuOpen(false);
    };
    const closeOnEscape = (event) => {
      if (event.key !== "Escape") return;
      setExportMenuOpen(false);
      exportMenuRef.current?.querySelector(".export-menu-trigger")?.focus();
    };

    document.addEventListener("pointerdown", closeOnOutsideClick);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsideClick);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [exportMenuOpen]);

  function changeResume(updater) {
    dirty.current = true;
    setSaveStatus("dirty");
    setResume((current) => typeof updater === "function" ? updater(current) : updater);
  }

  function changeDocument(nextDocument) {
    changeResume((current) => ({
      ...current,
      document: typeof nextDocument === "function" ? nextDocument(current.document) : nextDocument,
    }));
  }

  function selectSection(section, scroll = true) {
    if (!section) return;
    setSelectedSection(section);
    setPanelView("section");
    if (!scroll || (resume?.document.hidden_sections || []).includes(section)) return;
    window.requestAnimationFrame(() => {
      const workspace = previewRef.current;
      const target = workspace?.querySelector(`[data-section="${section}"]`);
      if (!workspace || !target) return;
      if (section === "basics" || section === "summary") {
        workspace.scrollTo({ top: 0, behavior: "smooth" });
        return;
      }
      const nextTop = workspace.scrollTop + target.getBoundingClientRect().top - workspace.getBoundingClientRect().top - 84;
      workspace.scrollTo({ top: Math.max(0, nextTop), behavior: "smooth" });
    });
  }

  async function performReview(session = aiSession) {
    let deterministicCompleted = false;
    const rulesStartedAt = Date.now();
    setReviewError("");
    setReviewStatus("loading");
    setReviewPhase("rules");
    try {
      const payload = { resume: resume.document, target_role: resume.document.basics?.headline || "" };
      const deterministic = await api.evaluateQuality(payload);
      deterministicCompleted = true;
      setQuality(deterministic);
      await keepPhaseVisible(rulesStartedAt, MIN_RULE_PROGRESS_MS);
      if (!session?.configured) {
        setReview(null);
        setReviewStatus("complete");
        return;
      }

      setReviewPhase("ai");
      const result = await api.reviewResume(payload);
      const validationStartedAt = Date.now();
      setReviewPhase("validate");
      if (result.quality) setQuality(result.quality);
      setReview(result);
      await keepPhaseVisible(validationStartedAt, MIN_VALIDATION_PROGRESS_MS);
      setReviewStatus("complete");
    } catch (error) {
      const hasFallback = deterministicCompleted || Boolean(quality);
      setReviewError(hasFallback ? `Forma checks completed, but AI feedback failed: ${error.message}` : error.message);
      setReviewStatus(hasFallback ? "complete" : "idle");
    }
  }

  async function runReview() {
    setPanelView("review");
    await performReview();
  }

  async function saveAISettings(form) {
    setSettingsSaving(true);
    setSettingsError("");
    try {
      const session = await api.setAISession(form);
      setAISession(session);
      setSettingsOpen(false);
      setToast("AI provider connected for this session");
      if (pendingReview) {
        setPendingReview(false);
        window.setTimeout(() => performReview(session), 0);
      }
    } catch (error) {
      setSettingsError(error.message);
    } finally {
      setSettingsSaving(false);
    }
  }

  async function clearAISettings() {
    setSettingsSaving(true);
    setSettingsError("");
    try {
      await api.clearAISession();
      setAISession({ configured: false });
      setReview(null);
      setReviewStatus("idle");
      setSettingsOpen(false);
      setPendingReview(false);
      setToast("AI provider disconnected");
    } catch (error) {
      setSettingsError(error.message);
    } finally {
      setSettingsSaving(false);
    }
  }

  async function duplicate() {
    try {
      const copy = await api.duplicateResume(id);
      navigate(`/resume/${copy.id}`);
    } catch (error) {
      setToast(error.message);
    }
  }

  async function downloadPDF() {
    if (exportingFormat) return;
    setExportMenuOpen(false);
    setExportingFormat("PDF");
    try {
      const { exportResumePDF } = await import("../lib/pdfExport.jsx");
      await exportResumePDF(resume);
      setToast("Selectable PDF downloaded.");
    } catch (error) {
      setToast(`Could not export PDF: ${error.message}`);
    } finally {
      setExportingFormat("");
    }
  }

  async function downloadDOCX() {
    if (exportingFormat) return;
    setExportMenuOpen(false);
    setExportingFormat("DOCX");
    try {
      const { exportResumeDOCX } = await import("../lib/docxExport.js");
      await exportResumeDOCX(resume);
      setToast("Editable DOCX downloaded.");
    } catch (error) {
      setToast(`Could not export DOCX: ${error.message}`);
    } finally {
      setExportingFormat("");
    }
  }

  function downloadJSON() {
    setExportMenuOpen(false);
    try {
      exportResumeJSON(resume);
      setToast("FORMA JSON backup downloaded.");
    } catch (error) {
      setToast(`Could not export JSON: ${error.message}`);
    }
  }

  function handleExportMenuKeyDown(event) {
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const items = [...(exportMenuRef.current?.querySelectorAll('[role="menuitem"]') || [])];
    if (!items.length) return;
    const currentIndex = items.indexOf(document.activeElement);
    if (event.key === "Home") items[0].focus();
    else if (event.key === "End") items.at(-1).focus();
    else if (event.key === "ArrowDown") items[currentIndex < 0 ? 0 : (currentIndex + 1) % items.length].focus();
    else items[currentIndex < 0 ? items.length - 1 : (currentIndex - 1 + items.length) % items.length].focus();
  }

  async function remove() {
    if (!window.confirm("Delete this resume? This cannot be undone.")) return;
    try {
      await api.deleteResume(id);
      navigate("/");
    } catch (error) {
      setToast(error.message);
    }
  }

  function addSection(sectionId) {
    let next = resume.document;
    if (!next.section_order.includes(sectionId)) next = { ...next, section_order: [...next.section_order, sectionId] };
    if ((next.hidden_sections || []).includes(sectionId)) next = toggleSection(next, sectionId);
    if (Array.isArray(next[sectionId]) && next[sectionId].length === 0) next = addArrayItem(next, sectionId);
    changeDocument(next);
    setAddOpen(false);
    window.setTimeout(() => selectSection(sectionId), 0);
  }

  if (loading) return <div className="full-page-state"><Spinner label="Loading resume" /><p>Opening your resume…</p></div>;
  if (loadError || !resume) return <div className="full-page-state"><EmptyState icon={<WarningCircle size={30} />} title="Could not open this resume" description={loadError || "Resume not found"} action={<Button onClick={() => navigate("/")}>Back to dashboard</Button>} /></div>;

  const saveLabel = saveStatus === "saving" ? "Saving…" : saveStatus === "error" ? "Save failed" : saveStatus === "dirty" ? "Unsaved changes" : "Saved locally";
  const reviewPanel = <AIReviewPanel
    session={aiSession}
    status={reviewStatus}
    phase={reviewPhase}
    error={reviewError}
    review={review}
    quality={quality}
    onRun={runReview}
    onConfigure={() => { setPendingReview(true); setSettingsOpen(true); }}
    onApply={(suggestion) => {
      const result = applyReviewSuggestionWithResult(resume.document, suggestion);
      if (result.applied) {
        changeDocument(result.document);
        setReview((current) => ({ ...current, suggestions: current.suggestions.filter((item) => item !== suggestion) }));
        setToast("Change applied and queued for local save. Verify the facts before export.");
        return;
      }
      if (result.reason === "already_applied") {
        setReview((current) => ({ ...current, suggestions: current.suggestions.filter((item) => item !== suggestion) }));
        setToast("This change is already in the resume.");
        return;
      }
      setToast("Could not match this suggestion to the current text. Run AI review again.");
    }}
    onDismiss={(suggestion) => setReview((current) => ({ ...current, suggestions: current.suggestions.filter((item) => item !== suggestion) }))}
  />;

  return (
    <div className={`editor-shell ${previewMode ? "editor-shell--preview" : ""}`}>
      <header className="editor-topbar">
        <div className="editor-topbar__left"><Link className="wordmark" to="/">FORMA</Link><span className={`save-state save-state--${saveStatus}`}>{saveStatus === "error" ? <WarningCircle size={16} /> : <CloudCheck size={16} />}{saveLabel}</span></div>
        <div className="editor-title"><InlineEdit value={resume.title} label="Document name" onCommit={(title) => changeResume((current) => ({ ...current, title }))} /></div>
        <div className="editor-topbar__actions">
          {previewMode ? <Button variant="ghost" onClick={() => setPreviewMode(false)}><ArrowLeft size={16} />Back to editor</Button> : <>
            <button type="button" onClick={() => setTemplatesOpen(true)}>Templates</button>
            <button type="button" className="mobile-preview-trigger" onClick={() => setPreviewMode(true)}><Eye size={16} />Preview</button>
            <div className="more-menu-wrap"><IconButton label="More actions" onClick={() => { setExportMenuOpen(false); setMoreOpen(!moreOpen); }}><DotsThree size={21} /></IconButton>{moreOpen && <div className="topbar-menu"><button type="button" onClick={() => { setMoreOpen(false); setImportOpen(true); }}><FileArrowUp size={16} />Import resume</button><button type="button" onClick={duplicate}><Copy size={16} />Duplicate</button><button type="button" className="danger" onClick={remove}><Trash size={16} />Delete</button></div>}</div>
            <div className="export-menu-wrap" ref={exportMenuRef}>
              <Button
                className="export-menu-trigger"
                onClick={() => {
                  setMoreOpen(false);
                  setExportMenuOpen((open) => !open);
                }}
                onKeyDown={(event) => {
                  if (event.key !== "ArrowDown") return;
                  event.preventDefault();
                  setMoreOpen(false);
                  setExportMenuOpen(true);
                  window.requestAnimationFrame(() => exportMenuRef.current?.querySelector('[role="menuitem"]')?.focus());
                }}
                disabled={Boolean(exportingFormat)}
                aria-busy={Boolean(exportingFormat)}
                aria-haspopup="menu"
                aria-expanded={exportMenuOpen}
                aria-controls="resume-export-menu"
              >
                <DownloadSimple size={16} />
                <span>{exportingFormat ? `Preparing ${exportingFormat}...` : "Export"}</span>
                {!exportingFormat && <CaretDown size={13} weight="bold" aria-hidden="true" />}
              </Button>
              {exportMenuOpen && <div id="resume-export-menu" className="topbar-menu export-menu" role="menu" aria-label="Export resume" onKeyDown={handleExportMenuKeyDown}>
                <button type="button" role="menuitem" onClick={downloadPDF}><FilePdf size={18} /><span><strong>PDF</strong><small>Polished document with selectable text</small></span></button>
                <button type="button" role="menuitem" onClick={downloadDOCX}><FileDoc size={18} /><span><strong>DOCX</strong><small>Editable in Word and Google Docs</small></span></button>
                <button type="button" role="menuitem" onClick={downloadJSON}><FileCode size={18} /><span><strong>FORMA JSON</strong><small>Portable backup for re-importing later</small></span></button>
              </div>}
            </div>
          </>}
        </div>
      </header>

      <div className="editor-body">
        {!previewMode && <aside className="editor-console">
          <SectionRail
            document={resume.document}
            selected={panelView === "section" ? selectedSection : null}
            activeView={panelView}
            onSelect={selectSection}
            onAdd={() => setAddOpen(true)}
            onMove={(section, direction) => changeDocument(moveSection(resume.document, section, direction))}
            onToggle={(section) => changeDocument(toggleSection(resume.document, section))}
            onOpenReview={() => setPanelView("review")}
            onOpenTemplates={() => setTemplatesOpen(true)}
            onOpenSettings={() => setSettingsOpen(true)}
          />
          <section className="editor-form-pane" aria-label={panelView === "review" ? "AI review" : "Resume editor"}>
            {panelView === "review" ? reviewPanel : <SectionEditorPanel section={selectedSection} document={resume.document} onChange={changeDocument} onOpenReview={() => setPanelView("review")} />}
          </section>
        </aside>}

        <main className="document-workspace" ref={previewRef}>
          {!previewMode && <div className="preview-bar"><div><span>Live preview</span><small>Changes save automatically</small></div><button type="button" onClick={() => setTemplatesOpen(true)}>{resume.document.template || "editorial"} template</button></div>}
          <div className="document-stage">
            <ResumeDocument
              document={resume.document}
              selectedSection={previewMode ? null : selectedSection}
              onSelectSection={selectSection}
              onChange={changeDocument}
              previewMode
              selectable={!previewMode}
            />
          </div>
        </main>
      </div>

      <TemplatesDialog open={templatesOpen} value={resume.document.template} paperSize={resume.document.paper_size} onClose={() => setTemplatesOpen(false)} onChange={({ template, paper_size }) => changeDocument({ ...resume.document, template, paper_size })} />
      <ImportResumeDialog
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onPreview={(file) => api.previewImport(file)}
        onApply={({ preview, mode, linkedinURL }) => {
          changeResume((current) => applyImportedResume(current, preview.candidate, { mode, linkedinURL }));
          setImportOpen(false);
          setToast(mode === "replace" ? "Imported content replaced this resume. Review every field before export." : "Imported content merged without overwriting existing fields.");
        }}
      />
      <AISettingsDialog open={settingsOpen} providers={providers} current={aiSession} onClose={() => { setSettingsOpen(false); setPendingReview(false); }} onSave={saveAISettings} onClear={clearAISettings} saving={settingsSaving} error={settingsError} />

      <Dialog open={addOpen} onClose={() => setAddOpen(false)} title="Add a section" description="Sections can be reordered or hidden without losing their content.">
        <div className="add-section-grid">{sectionCatalog.filter((section) => !["basics", "summary"].includes(section.id)).map((section) => <button type="button" key={section.id} onClick={() => addSection(section.id)}><span>{section.label}</span><small>{resume.document.section_order.includes(section.id) ? "Show section" : "Add to document"}</small></button>)}</div>
      </Dialog>

      <Toast message={toast} onDismiss={() => setToast("")} />
    </div>
  );
}
