import { useEffect, useId, useState } from "react";
import {
  FileArrowUp,
  FileDoc,
  LinkedinLogo,
  LinkSimple,
  WarningCircle,
} from "@phosphor-icons/react";
import { Button, Dialog, Field, Spinner } from "./ui.jsx";

const ACCEPTED_RESUME_FILES = ".json,.zip,.pdf,.docx,application/json,application/pdf,application/zip,application/vnd.openxmlformats-officedocument.wordprocessingml.document";
const ACCEPTED_LINKEDIN_FILES = ".zip,.pdf,application/pdf,application/zip";

function isLinkedInProfileURL(value) {
  if (!value.trim()) return true;
  try {
    const url = new URL(value);
    return /(^|\.)linkedin\.com$/i.test(url.hostname) && /^\/in\//i.test(url.pathname);
  } catch {
    return false;
  }
}

function candidateSummary(candidate = {}) {
  const document = candidate.document || {};
  return [
    ["Experience", document.experience?.length || 0],
    ["Projects", document.projects?.length || 0],
    ["Education", document.education?.length || 0],
    ["Skill groups", document.skills?.length || 0],
  ];
}

export function ImportResumeDialog({ open, onClose, onPreview, onApply, allowMerge = true }) {
  const fileInputId = useId();
  const [source, setSource] = useState("file");
  const [file, setFile] = useState(null);
  const [linkedinURL, setLinkedinURL] = useState("");
  const [status, setStatus] = useState("idle");
  const [error, setError] = useState("");
  const [preview, setPreview] = useState(null);
  const [mode, setMode] = useState("merge");

  useEffect(() => {
    if (!open) return;
    setSource("file");
    setFile(null);
    setLinkedinURL("");
    setStatus("idle");
    setError("");
    setPreview(null);
    setMode(allowMerge ? "merge" : "replace");
  }, [allowMerge, open]);

  const validLinkedInURL = isLinkedInProfileURL(linkedinURL);

  async function inspectFile() {
    if (!file) return;
    setStatus("loading");
    setError("");
    try {
      const result = await onPreview(file);
      setPreview(result);
      setStatus("ready");
    } catch (nextError) {
      setPreview(null);
      setStatus("idle");
      setError(nextError.message || "Forma could not read this file.");
    }
  }

  function chooseFile(event) {
    const nextFile = event.target.files?.[0] || null;
    event.target.value = "";
    setFile(nextFile);
    setPreview(null);
    setError("");
    setStatus("idle");
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      className="dialog--import"
      title="Import a resume"
      description="Forma previews every field before it changes your current resume. Uploaded files are parsed in memory and are not stored."
    >
      <div className="import-source-tabs" role="tablist" aria-label="Import source">
        <button type="button" role="tab" aria-selected={source === "file"} onClick={() => { setSource("file"); setFile(null); setPreview(null); setError(""); }}>
          <FileDoc size={18} /> Resume file
        </button>
        <button type="button" role="tab" aria-selected={source === "linkedin"} onClick={() => { setSource("linkedin"); setFile(null); setPreview(null); setError(""); }}>
          <LinkedinLogo size={18} /> LinkedIn
        </button>
      </div>

      {source === "linkedin" && (
        <div className="linkedin-import-note">
          <div className="linkedin-import-note__icon"><LinkedinLogo size={22} /></div>
          <div>
            <strong>Use your official LinkedIn export</strong>
            <p>LinkedIn blocks reliable profile scraping. Paste your profile URL so Forma can keep it as a contact link, then upload your data ZIP or “Save to PDF” export.</p>
            <a href="https://www.linkedin.com/help/linkedin/answer/a1339364" target="_blank" rel="noreferrer">How to download your LinkedIn data</a>
          </div>
        </div>
      )}

      {source === "linkedin" && (
        <Field label="LinkedIn profile URL" hint={validLinkedInURL ? "Optional. It will be added to the imported contact links." : "Use a full linkedin.com/in/... profile URL."}>
          <div className={`import-url-field ${validLinkedInURL ? "" : "is-invalid"}`}>
            <LinkSimple size={17} />
            <input value={linkedinURL} onChange={(event) => setLinkedinURL(event.target.value)} placeholder="https://www.linkedin.com/in/your-profile" aria-invalid={!validLinkedInURL} />
          </div>
        </Field>
      )}

      <div className="import-dropzone">
        <FileArrowUp size={28} />
        <strong>{file ? file.name : source === "linkedin" ? "Choose a LinkedIn ZIP or PDF" : "Choose a resume file"}</strong>
        <p>{file ? `${Math.max(1, Math.round(file.size / 1024))} KB ready to inspect` : source === "linkedin" ? "Official data archive ZIP or text-based profile PDF" : "Forma JSON, JSON Resume, PDF, or DOCX · up to 12 MB"}</p>
        <label className="button button--secondary button--md" htmlFor={fileInputId}>{file ? "Choose another file" : "Browse files"}</label>
        <input id={fileInputId} type="file" hidden accept={source === "linkedin" ? ACCEPTED_LINKEDIN_FILES : ACCEPTED_RESUME_FILES} onChange={chooseFile} />
      </div>

      {error && <div className="form-error" role="alert"><WarningCircle size={17} />{error}</div>}

      {!preview ? (
        <div className="dialog__actions">
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button onClick={inspectFile} disabled={!file || !validLinkedInURL || status === "loading"}>
            {status === "loading" ? <><Spinner label="Inspecting resume" /> Inspecting file</> : "Preview import"}
          </Button>
        </div>
      ) : (
        <div className="import-preview" aria-live="polite">
          <div className="import-preview__header">
            <div><span>Import preview</span><strong>{preview.candidate?.basics?.full_name || preview.candidate?.basics?.name || preview.candidate?.document?.basics?.full_name || preview.candidate?.document?.basics?.name || preview.candidate?.title || "Imported resume"}</strong></div>
            <small>{preview.parser?.id || preview.parser || "detected format"}{preview.parser?.version ? ` · v${preview.parser.version}` : ""}</small>
          </div>
          <div className="import-preview__stats">
            {candidateSummary(preview.candidate).map(([label, value]) => <div key={label}><strong>{value}</strong><span>{label}</span></div>)}
          </div>
          {(preview.warnings || []).length > 0 && (
            <div className="import-preview__warnings">
              {(preview.warnings || []).map((warning, index) => <p key={warning.code || `${warning}-${index}`}><WarningCircle size={15} />{warning.message || warning}</p>)}
            </div>
          )}
          {allowMerge && <fieldset className="import-mode">
            <legend>How should Forma apply it?</legend>
            <label><input type="radio" name="import-mode" value="merge" checked={mode === "merge"} onChange={() => setMode("merge")} /><span><strong>Merge safely</strong><small>Fill empty fields and add new entries without overwriting your current content.</small></span></label>
            <label><input type="radio" name="import-mode" value="replace" checked={mode === "replace"} onChange={() => setMode("replace")} /><span><strong>Replace resume content</strong><small>Use the imported content but keep your current template and page settings.</small></span></label>
          </fieldset>}
          <div className="dialog__actions">
            <Button variant="ghost" onClick={() => { setPreview(null); setStatus("idle"); }}>Back</Button>
            <Button onClick={() => onApply({ preview, mode, linkedinURL: validLinkedInURL ? linkedinURL.trim() : "" })}>Apply import</Button>
          </div>
        </div>
      )}
    </Dialog>
  );
}

export { isLinkedInProfileURL };
