import { emptyResumeDocument, sectionCatalog } from "../data/sampleResume.js";

export function clone(value) {
  return structuredClone(value);
}

export function uid(prefix = "item") {
  return `${prefix}-${crypto.randomUUID()}`;
}

export function formatPeriod(startDate, endDate, current = false, locale = "en") {
  const format = (value) => {
    if (!value) return "";
    const [year, month] = String(value).split("-");
    if (!month) return year;
    const date = new Date(Number(year), Number(month) - 1, 1);
    return new Intl.DateTimeFormat(locale, { month: "short", year: "numeric" }).format(date);
  };
  const start = format(startDate);
  const end = current ? "Present" : format(endDate);
  return [start, end].filter(Boolean).join(" — ");
}

export function fileSafeName(value, extension = "json") {
  const base = String(value || "resume")
    .normalize("NFKD")
    .replace(/[^a-zA-Z0-9\s-_]/g, "")
    .trim()
    .replace(/[\s_]+/g, "-")
    .toLowerCase() || "resume";
  return `${base}.${extension}`;
}

export function exportResumeJSON(resume) {
  const blob = new Blob([JSON.stringify({ format: "forma.resume", version: 1, resume }, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileSafeName(resume.title);
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export async function parseResumeFile(file) {
  if (!file || file.size > 2_000_000) throw new Error("Choose a JSON file smaller than 2 MB.");
  let parsed;
  try {
    parsed = JSON.parse(await file.text());
  } catch {
    throw new Error("This file is not valid JSON.");
  }
  const candidate = parsed?.format === "forma.resume" ? parsed.resume : parsed;
  if (!candidate || typeof candidate !== "object" || typeof candidate.title !== "string" || !candidate.document) {
    throw new Error("This does not look like a Forma resume export.");
  }
  return {
    title: candidate.title.slice(0, 160),
    document: normalizeDocument(candidate.document),
  };
}

export function normalizeDocument(input) {
  const source = input && typeof input === "object" ? input : {};
  const normalized = {
    ...clone(emptyResumeDocument),
    ...source,
    basics: { ...emptyResumeDocument.basics, ...(source.basics || {}) },
  };
  for (const field of ["experience", "projects", "portfolio", "education", "skills", "certifications", "languages", "custom_sections"]) {
    normalized[field] = Array.isArray(source[field]) ? source[field] : [];
  }
  const allowedSections = new Set([...sectionCatalog.map((section) => section.id), ...normalized.custom_sections.map((section) => section.id)]);
  const order = Array.isArray(source.section_order) ? source.section_order.filter((section) => allowedSections.has(section)) : [];
  normalized.section_order = [...new Set([...order, ...emptyResumeDocument.section_order])];
  normalized.hidden_sections = Array.isArray(source.hidden_sections) ? source.hidden_sections : [];
  return normalized;
}

const importArraySections = ["experience", "projects", "portfolio", "education", "skills", "certifications", "languages", "custom_sections"];

function importKey(section, item = {}) {
  const partsBySection = {
    experience: [item.company, item.position, item.start_date, item.end_date],
    projects: [item.name, item.url, item.start_date],
    portfolio: [item.name, item.url],
    education: [item.institution, item.degree ?? item.study_type, item.area, item.end_date],
    skills: [item.name],
    certifications: [item.name, item.issuer, item.date],
    languages: [item.language ?? item.name],
    custom_sections: [item.title, item.id],
  };
  return (partsBySection[section] || [item.id])
    .map((value) => String(value || "").normalize("NFKC").trim().toLocaleLowerCase())
    .filter(Boolean)
    .join("|");
}

function addLinkedInProfile(document, linkedinURL) {
  if (!linkedinURL) return document;
  const profiles = [...(document.basics?.profiles || [])];
  if (!profiles.some((profile) => String(profile.url || "").replace(/\/$/, "") === linkedinURL.replace(/\/$/, ""))) {
    profiles.push({ id: uid("profile"), network: "LinkedIn", username: "", url: linkedinURL });
  }
  return { ...document, basics: { ...document.basics, profiles } };
}

export function applyImportedResume(currentResume, candidate, { mode = "merge", linkedinURL = "" } = {}) {
  const currentDocument = normalizeDocument(currentResume?.document);
  let importedDocument = addLinkedInProfile(normalizeDocument(candidate?.document), linkedinURL);

  if (mode === "replace") {
    importedDocument = {
      ...importedDocument,
      template: currentDocument.template,
      paper_size: currentDocument.paper_size,
      language: importedDocument.language || currentDocument.language,
      hidden_sections: currentDocument.hidden_sections,
    };
    return { ...currentResume, document: importedDocument };
  }

  const mergedBasics = { ...currentDocument.basics };
  for (const field of ["full_name", "headline", "email", "phone", "location", "website", "photo_url"]) {
    const currentValue = String(mergedBasics[field] || "").trim();
    const isPlaceholder = field === "full_name" && currentValue === "Your name";
    if ((!currentValue || isPlaceholder) && String(importedDocument.basics?.[field] || "").trim()) mergedBasics[field] = importedDocument.basics[field];
  }
  mergedBasics.profiles = [...(currentDocument.basics?.profiles || [])];
  for (const profile of importedDocument.basics?.profiles || []) {
    const profileURL = String(profile.url || "").replace(/\/$/, "");
    if (profileURL && !mergedBasics.profiles.some((current) => String(current.url || "").replace(/\/$/, "") === profileURL)) {
      mergedBasics.profiles.push({ ...profile, id: profile.id || uid("profile") });
    }
  }

  const merged = {
    ...currentDocument,
    basics: mergedBasics,
    summary: currentDocument.summary?.trim() ? currentDocument.summary : importedDocument.summary,
    section_order: [...new Set([...(currentDocument.section_order || []), ...(importedDocument.section_order || [])])],
  };
  for (const section of importArraySections) {
    const existing = [...(currentDocument[section] || [])];
    const keys = new Set(existing.map((item) => importKey(section, item)).filter(Boolean));
    for (const item of importedDocument[section] || []) {
      const key = importKey(section, item);
      if (key && keys.has(key)) continue;
      existing.push({ ...item, id: item.id || uid(section.replace(/s$/, "")) });
      if (key) keys.add(key);
    }
    merged[section] = existing;
  }
  return { ...currentResume, document: normalizeDocument(merged) };
}

export function moveSection(document, sectionId, direction) {
  const next = clone(document);
  const index = next.section_order.indexOf(sectionId);
  const target = index + direction;
  if (index < 0 || target < 0 || target >= next.section_order.length) return next;
  [next.section_order[index], next.section_order[target]] = [next.section_order[target], next.section_order[index]];
  return next;
}

export function toggleSection(document, sectionId) {
  const next = clone(document);
  const hidden = new Set(next.hidden_sections || []);
  if (hidden.has(sectionId)) hidden.delete(sectionId);
  else hidden.add(sectionId);
  next.hidden_sections = [...hidden];
  return next;
}

export function updateArrayItem(document, section, itemId, updater) {
  return {
    ...document,
    [section]: (document[section] || []).map((item) => (item.id === itemId ? updater({ ...item }) : item)),
  };
}

export function removeArrayItem(document, section, itemId) {
  return { ...document, [section]: (document[section] || []).filter((item) => item.id !== itemId) };
}

export function addArrayItem(document, section) {
  const factories = {
    experience: () => ({ id: uid("exp"), company: "New company", position: "Role", location: "", start_date: "", end_date: "", current: false, summary: "", highlights: ["Describe a verified outcome."], skills: [] }),
    projects: () => ({ id: uid("project"), name: "New project", role: "", start_date: "", end_date: "", description: "", highlights: [], technologies: [], url: "" }),
    portfolio: () => ({ id: uid("portfolio"), name: "Portfolio item", description: "Describe the work and your contribution.", url: "" }),
    education: () => ({ id: uid("edu"), institution: "Institution", degree: "Degree", area: "Field of study", location: "", start_date: "", end_date: "", score: "", highlights: [] }),
    skills: () => ({ id: uid("skill"), name: "Skill group", items: ["Skill"] }),
    certifications: () => ({ id: uid("cert"), name: "Certification", issuer: "Issuer", date: "", expiry_date: "", credential_id: "", url: "" }),
    languages: () => ({ id: uid("lang"), language: "Language", fluency: "B2" }),
  };
  const factory = factories[section];
  if (!factory) return document;
  return { ...document, [section]: [...(document[section] || []), factory()] };
}

export function resumeReadiness(document) {
  const checks = [
    { id: "name", label: "Name added", pass: Boolean(document.basics?.full_name && document.basics.full_name !== "Your name") },
    { id: "contact", label: "Contact method added", pass: Boolean(document.basics?.email || document.basics?.phone || document.basics?.website) },
    { id: "experience", label: "Experience has outcomes", pass: (document.experience || []).some((item) => (item.highlights || []).some((line) => /\d|%|increase|reduce|save|grow/i.test(line))) },
    { id: "summary", label: "Summary is concise", pass: Boolean(document.summary && document.summary.length >= 40 && document.summary.length <= 600) },
  ];
  return { checks, passed: checks.filter((check) => check.pass).length, total: checks.length };
}

const aiSectionAliases = {
  header: "basics",
  contact: "basics",
  personal: "basics",
  work: "experience",
  work_experience: "experience",
  professional_experience: "experience",
  employment: "experience",
  project: "projects",
  skill: "skills",
  certification: "certifications",
  language: "languages",
};

const aiEditableFields = {
  basics: ["headline"],
  experience: ["position", "company", "summary", "highlights"],
  projects: ["name", "role", "description", "highlights", "technologies"],
  portfolio: ["name", "description"],
  education: ["institution", "degree", "area", "highlights"],
  skills: ["name", "items"],
  certifications: ["name", "issuer"],
  languages: ["language", "fluency"],
};

const aiFieldAliases = {
  basics: { name: "full_name" },
  projects: { summary: "description", keywords: "technologies" },
  education: { study_type: "degree" },
  skills: { keywords: "items" },
  languages: { name: "language" },
};

function normalizeAISection(value) {
  const section = String(value || "").trim().toLowerCase().replace(/[\s-]+/g, "_");
  return aiSectionAliases[section] || section;
}

function normalizeAIField(section, value) {
  const field = String(value || "").trim().toLowerCase();
  return aiFieldAliases[section]?.[field] || field;
}

function comparableText(value) {
  return String(value || "")
    .normalize("NFKC")
    .toLocaleLowerCase()
    .replace(/^[\s"'`“”‘’•*-]+|[\s"'`“”‘’]+$/gu, "")
    .replace(/[^\p{L}\p{N}%+]+/gu, " ")
    .trim();
}

function matchSuggestionText(source, original, replacement) {
  const sourceText = String(source || "");
  const originalText = String(original || "").trim().replace(/^["'`“”‘’]+|["'`“”‘’]+$/gu, "");
  if (!sourceText || !originalText) return null;
  if (sourceText === originalText) return { score: 100, value: replacement };

  const exactIndex = sourceText.indexOf(originalText);
  if (exactIndex >= 0) {
    return { score: 98, value: `${sourceText.slice(0, exactIndex)}${replacement}${sourceText.slice(exactIndex + originalText.length)}` };
  }

  const foldedIndex = sourceText.toLocaleLowerCase().indexOf(originalText.toLocaleLowerCase());
  if (foldedIndex >= 0) {
    return { score: 96, value: `${sourceText.slice(0, foldedIndex)}${replacement}${sourceText.slice(foldedIndex + originalText.length)}` };
  }

  if (comparableText(sourceText) === comparableText(originalText)) return { score: 94, value: replacement };
  return null;
}

function collectSuggestionTargets(document, section, suggestion) {
  if (section === "summary") return [{ section, field: "summary", value: document.summary || "" }];
  if (section === "basics") {
    const fields = suggestion.field ? [normalizeAIField(section, suggestion.field)] : aiEditableFields.basics;
    return fields.filter((field) => aiEditableFields.basics.includes(field) && typeof document.basics?.[field] === "string")
      .map((field) => ({ section, field, value: document.basics[field] }));
  }

  if (!Array.isArray(document[section]) || !aiEditableFields[section]) return [];
  const requestedField = suggestion.field ? normalizeAIField(section, suggestion.field) : "";
  const allowedFields = requestedField && aiEditableFields[section].includes(requestedField) ? [requestedField] : aiEditableFields[section];
  const targets = [];
  document[section].forEach((item, itemIndex) => {
    if (suggestion.item_id && item.id !== suggestion.item_id) return;
    for (const field of allowedFields) {
      const value = item[field];
      if (typeof value === "string") targets.push({ section, itemIndex, itemId: item.id, field, value });
      if (Array.isArray(value)) value.forEach((entry, arrayIndex) => {
        if (typeof entry === "string" && (!Number.isInteger(suggestion.index) || suggestion.index === arrayIndex)) {
          targets.push({ section, itemIndex, itemId: item.id, field, arrayIndex, value: entry });
        }
      });
    }
  });
  return targets;
}

function setSuggestionTarget(document, target, value) {
  if (target.section === "summary") return { ...document, summary: value };
  if (target.section === "basics") return { ...document, basics: { ...document.basics, [target.field]: value } };
  const items = [...(document[target.section] || [])];
  const item = { ...items[target.itemIndex] };
  if (Number.isInteger(target.arrayIndex)) {
    const values = [...(item[target.field] || [])];
    values[target.arrayIndex] = value;
    item[target.field] = values;
  } else {
    item[target.field] = value;
  }
  items[target.itemIndex] = item;
  return { ...document, [target.section]: items };
}

export function applyReviewSuggestionWithResult(document, suggestion) {
  const section = normalizeAISection(suggestion?.section);
  const proposed = suggestion?.proposed ?? suggestion?.replacement;
  if (!section || typeof proposed !== "string" || !proposed.trim() || !String(suggestion?.original || "").trim()) {
    return { document, applied: false, reason: "invalid_suggestion" };
  }

  const targets = collectSuggestionTargets(document, section, suggestion);
  if (!targets.length) return { document, applied: false, reason: "unsupported_target", section };

  const directTarget = suggestion.field && (section === "summary" || suggestion.item_id || section === "basics")
    ? targets[0]
    : null;
  if (directTarget) {
    if (directTarget.value === proposed) return { document, applied: false, reason: "already_applied", section, target: directTarget };
    const match = matchSuggestionText(directTarget.value, suggestion.original, proposed);
    if (!match) return { document, applied: false, reason: "source_not_found", section, target: directTarget };
    return { document: setSuggestionTarget(document, directTarget, match.value), applied: true, section, target: directTarget };
  }

  const matches = targets
    .map((target) => ({ target, match: matchSuggestionText(target.value, suggestion.original, proposed) }))
    .filter(({ match }) => match)
    .sort((left, right) => right.match.score - left.match.score);

  if (!matches.length) {
    const alreadyApplied = targets.find((target) => comparableText(target.value) === comparableText(proposed));
    return alreadyApplied
      ? { document, applied: false, reason: "already_applied", section, target: alreadyApplied }
      : { document, applied: false, reason: "source_not_found", section };
  }
  if (matches[1] && matches[0].match.score === matches[1].match.score) {
    return { document, applied: false, reason: "ambiguous_source", section };
  }
  if (matches[0].match.value === matches[0].target.value) {
    return { document, applied: false, reason: "already_applied", section, target: matches[0].target };
  }
  return {
    document: setSuggestionTarget(document, matches[0].target, matches[0].match.value),
    applied: true,
    section,
    target: matches[0].target,
  };
}

export function applyReviewSuggestion(document, suggestion) {
  return applyReviewSuggestionWithResult(document, suggestion).document;
}
