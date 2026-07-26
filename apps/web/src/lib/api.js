import { makeSampleResume } from "../data/sampleResume.js";

const API_ROOT = (import.meta.env.VITE_API_URL || "/api/v1").replace(/\/$/, "");
const USE_MOCK_API = import.meta.env.VITE_USE_MOCK_API === "true";
const MOCK_STORAGE_KEY = "forma.mock.resumes.v1";

function toApiDocument(document = {}) {
  const basics = document.basics || {};
  return {
    version: document.version || 1,
    basics: {
      name: basics.full_name ?? basics.name ?? "",
      headline: basics.headline || "",
      email: basics.email || "",
      phone: basics.phone || "",
      location: basics.location || "",
      website: basics.website || "",
      photo_url: basics.photo_url || "",
      links: (basics.profiles || basics.links || []).map((item) => ({
        label: item.network || item.label || "Link",
        url: item.url || "",
      })),
    },
    summary: document.summary || "",
    experience: (document.experience || []).map(({ id, company, position, location, start_date, end_date, current, summary, highlights }) => ({ id, company, position, location, start_date, end_date, current, summary, highlights })),
    projects: (document.projects || []).map((item) => ({
      id: item.id,
      name: item.name || "",
      role: item.role || "",
      url: item.url || "",
      start_date: item.start_date || "",
      end_date: item.end_date || "",
      summary: item.description ?? item.summary ?? "",
      highlights: item.highlights || [],
      keywords: item.technologies ?? item.keywords ?? [],
    })),
    education: (document.education || []).map((item) => ({
      id: item.id,
      institution: item.institution || "",
      study_type: item.degree ?? item.study_type ?? "",
      area: item.area || "",
      location: item.location || "",
      start_date: item.start_date || "",
      end_date: item.end_date || "",
      score: item.score || "",
      highlights: item.highlights || [],
    })),
    skills: (document.skills || []).map((item) => ({ id: item.id, name: item.name || "", keywords: item.items ?? item.keywords ?? [], level: item.level || "" })),
    portfolio: (document.portfolio || []).map(({ id, name, description, url }) => ({ id, name, description, url })),
    certifications: (document.certifications || []).map(({ id, name, issuer, date, url }) => ({ id, name, issuer, date, url })),
    languages: (document.languages || []).map((item) => ({ id: item.id, name: item.language ?? item.name ?? "", fluency: item.fluency || "" })),
    custom_sections: document.custom_sections || [],
    order: document.section_order ?? document.order ?? [],
    hidden_sections: document.hidden_sections || [],
    template: document.template || "editorial",
    page_size: String(document.paper_size ?? document.page_size ?? "a4").toUpperCase(),
    language: document.language || "en",
  };
}

function fromApiDocument(document = {}) {
  const basics = document.basics || {};
  return {
    version: document.version || 1,
    basics: {
      full_name: basics.name ?? basics.full_name ?? "",
      headline: basics.headline || "",
      email: basics.email || "",
      phone: basics.phone || "",
      location: basics.location || "",
      website: basics.website || "",
      photo_url: basics.photo_url || "",
      profiles: (basics.links || basics.profiles || []).map((item, index) => ({
        id: item.id || `profile-${index}`,
        network: item.label || item.network || "Link",
        username: item.username || "",
        url: item.url || "",
      })),
    },
    summary: document.summary || "",
    experience: document.experience || [],
    projects: (document.projects || []).map((item) => ({ ...item, description: item.summary ?? item.description ?? "", technologies: item.keywords ?? item.technologies ?? [] })),
    education: (document.education || []).map((item) => ({ ...item, degree: item.study_type ?? item.degree ?? "" })),
    skills: (document.skills || []).map((item) => ({ ...item, items: item.keywords ?? item.items ?? [] })),
    portfolio: document.portfolio || [],
    certifications: document.certifications || [],
    languages: (document.languages || []).map((item) => ({ ...item, language: item.name ?? item.language ?? "" })),
    custom_sections: document.custom_sections || [],
    section_order: document.order ?? document.section_order ?? [],
    hidden_sections: document.hidden_sections || [],
    template: document.template || "editorial",
    paper_size: String(document.page_size ?? document.paper_size ?? "A4").toLowerCase(),
    language: document.language || "en",
  };
}

function toApiResume(payload) {
  return { ...payload, document: toApiDocument(payload.document) };
}

function fromApiResume(resume) {
  return resume?.document ? { ...resume, document: fromApiDocument(resume.document) } : resume;
}

function normalizeProvider(provider) {
  return {
    ...provider,
    suggested_model: provider.suggested_model ?? provider.default_model ?? provider.suggested_models?.[0] ?? "",
    base_url: provider.base_url ?? provider.default_base_url ?? "",
    key_required: provider.key_required ?? provider.requires_api_key ?? false,
    base_url_editable: provider.base_url_editable ?? provider.supports_custom_base_url ?? false,
  };
}

function normalizeReview(result) {
  const review = result?.ai || result;
  return {
    ...review,
    quality: result?.quality || review?.quality || null,
    suggestions: (review?.suggestions || []).map((suggestion) => ({
      ...suggestion,
      proposed: suggestion.proposed ?? suggestion.replacement ?? "",
    })),
  };
}

export class ApiError extends Error {
  constructor(message, { status = 0, code = "request_failed", details = null } = {}) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

async function request(path, options = {}) {
  const isMultipart = typeof FormData !== "undefined" && options.body instanceof FormData;
  const response = await fetch(`${API_ROOT}${path}`, {
    credentials: "include",
    headers: {
      Accept: "application/json",
      ...(options.body && !isMultipart ? { "Content-Type": "application/json" } : {}),
      ...options.headers,
    },
    ...options,
  });

  const body = response.status === 204 ? null : await response.json().catch(() => null);
  if (!response.ok) {
    const problem = body?.error || body;
    throw new ApiError(problem?.message || `Request failed (${response.status})`, {
      status: response.status,
      code: problem?.code || "request_failed",
      details: problem?.details ?? problem?.fields ?? null,
    });
  }
  return body?.data ?? body;
}

function readMockResumes() {
  try {
    const stored = JSON.parse(localStorage.getItem(MOCK_STORAGE_KEY) || "[]");
    if (Array.isArray(stored) && stored.length) return stored;
  } catch {
    // Ignore malformed developer-only mock data.
  }
  const first = {
    id: crypto.randomUUID(),
    ...makeSampleResume(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
  localStorage.setItem(MOCK_STORAGE_KEY, JSON.stringify([first]));
  return [first];
}

function writeMockResumes(resumes) {
  localStorage.setItem(MOCK_STORAGE_KEY, JSON.stringify(resumes));
}

let mockAIConfig = null;

const providerPresets = [
  { id: "openai", name: "OpenAI", protocol: "OpenAI Responses", suggested_model: "gpt-5.6-terra", base_url: "https://api.openai.com/v1", structured_output: "json_schema", key_required: true },
  { id: "anthropic", name: "Anthropic", protocol: "Anthropic Messages", suggested_model: "claude-sonnet-5", base_url: "https://api.anthropic.com/v1", structured_output: "json_schema", key_required: true },
  { id: "gemini", name: "Google Gemini", protocol: "Gemini native", suggested_model: "gemini-3-flash", base_url: "https://generativelanguage.googleapis.com/v1beta", structured_output: "json_schema", key_required: true },
  { id: "deepseek", name: "DeepSeek", protocol: "OpenAI Chat Completions", suggested_model: "deepseek-v4-flash", base_url: "https://api.deepseek.com", structured_output: "json_object", key_required: true },
  { id: "qwen", name: "Qwen / DashScope", protocol: "OpenAI Chat Completions", suggested_model: "qwen-plus", base_url: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", structured_output: "json_object", key_required: true, base_url_editable: true },
  { id: "kimi", name: "Kimi / Moonshot", protocol: "OpenAI Chat Completions", suggested_model: "kimi-k3", base_url: "https://api.moonshot.ai/v1", structured_output: "json_schema", key_required: true, base_url_editable: true },
  { id: "glm", name: "GLM / Z.AI", protocol: "OpenAI Chat Completions", suggested_model: "glm-5.1", base_url: "https://api.z.ai/api/paas/v4", structured_output: "json_object", key_required: true, base_url_editable: true },
  { id: "ollama", name: "Ollama", protocol: "Local OpenAI-compatible", suggested_model: "qwen3", base_url: "http://host.docker.internal:11434/v1", structured_output: "json_schema", key_required: false, base_url_editable: true, local: true },
  { id: "custom", name: "Custom endpoint", protocol: "OpenAI Chat Completions", suggested_model: "", base_url: "", structured_output: "prompt_only", key_required: false, base_url_editable: true },
];

const mock = {
  async health() {
    return { status: "ok", mode: "mock" };
  },
  async listResumes() {
    return readMockResumes().map(({ document, ...resume }) => ({
      ...resume,
      target_role: document?.basics?.headline || "",
      template: document?.template || "editorial",
    }));
  },
  async createResume(payload) {
    const resumes = readMockResumes();
    const now = new Date().toISOString();
    const resume = { id: crypto.randomUUID(), ...payload, created_at: now, updated_at: now };
    writeMockResumes([resume, ...resumes]);
    return resume;
  },
  async getResume(id) {
    const resume = readMockResumes().find((item) => item.id === id);
    if (!resume) throw new ApiError("Resume not found", { status: 404, code: "not_found" });
    return structuredClone(resume);
  },
  async updateResume(id, payload) {
    const resumes = readMockResumes();
    const index = resumes.findIndex((item) => item.id === id);
    if (index < 0) throw new ApiError("Resume not found", { status: 404, code: "not_found" });
    resumes[index] = { ...resumes[index], ...payload, id, updated_at: new Date().toISOString() };
    writeMockResumes(resumes);
    return structuredClone(resumes[index]);
  },
  async deleteResume(id) {
    writeMockResumes(readMockResumes().filter((item) => item.id !== id));
  },
  async duplicateResume(id) {
    const source = await this.getResume(id);
    const now = new Date().toISOString();
    const copy = { ...structuredClone(source), id: crypto.randomUUID(), title: `${source.title} copy`, created_at: now, updated_at: now };
    writeMockResumes([copy, ...readMockResumes()]);
    return copy;
  },
  async providers() {
    return providerPresets;
  },
  async getAISession() {
    return mockAIConfig ? { configured: true, provider: mockAIConfig.provider, model: mockAIConfig.model, base_url: mockAIConfig.base_url } : { configured: false };
  },
  async setAISession(config) {
    mockAIConfig = { ...config };
    return { configured: true, provider: config.provider, model: config.model, base_url: config.base_url };
  },
  async clearAISession() {
    mockAIConfig = null;
  },
  async reviewResume() {
    if (!mockAIConfig) throw new ApiError("Choose an AI provider first", { status: 409, code: "ai_not_configured" });
    await new Promise((resolve) => setTimeout(resolve, 650));
    return {
      summary: "Strong, concise draft. Two bullets can make the outcome clearer.",
      assessments: [
        { rule_id: "semantic.impact_strength", verdict: "partial", evidence: "Built alerting workflow that reduced noise and improved response time.", confidence: 0.9, reason: "The action is clear but the verified outcome could be more specific." },
        { rule_id: "semantic.clarity_specificity", verdict: "pass", evidence: "Product engineer with 6+ years building user-centered SaaS products.", confidence: 0.92, reason: "The positioning is concise and specific." },
      ],
      suggestions: [
        { id: "mock-1", section: "experience", item_id: "exp-sentry", field: "highlights", index: 0, severity: "improve", title: "Show the measurable result", reason: "The bullet explains the work but not its verified outcome.", original: "Built alerting workflow that reduced noise and improved response time.", proposed: "Built an alerting workflow that reduced false alerts and improved mean response time." },
        { id: "mock-2", section: "summary", field: "summary", severity: "note", title: "Name the target role", reason: "A role-specific opening improves relevance without adding unsupported facts.", original: "Product engineer with 6+ years building user-centered SaaS products.", proposed: "Product engineer with 6+ years building user-centered SaaS products." },
      ],
      checks: { clarity: "pass", ats: "pass", unsupported_claims: "review" },
    };
  },
  async evaluateQuality() {
    return {
      rubric_version: "forma-quality/1.0.0",
      source_digest: "mock",
      language: "en",
      quality: { score: 49, maximum_score: 100, assessed_points: 60, unassessed_points: 40, normalized_score: 82, ready: true, blockers: [], categories: [
        { id: "essentials", label: "Essentials and focus", maximum_points: 15, assessed_points: 15, earned_points: 15, unassessed_points: 0, status: "pass" },
        { id: "structure", label: "Structure and completeness", maximum_points: 10, assessed_points: 10, earned_points: 8, unassessed_points: 0, status: "warn" },
        { id: "evidence", label: "Evidence signals", maximum_points: 15, assessed_points: 15, earned_points: 11, unassessed_points: 0, status: "warn" },
        { id: "clarity", label: "Clarity mechanics", maximum_points: 12, assessed_points: 12, earned_points: 9, unassessed_points: 0, status: "warn" },
        { id: "consistency", label: "Consistency and chronology", maximum_points: 8, assessed_points: 8, earned_points: 6, unassessed_points: 0, status: "warn" },
      ] },
      ats_hygiene: { status: "pass", findings: [] },
      semantic: { maximum_points: 40, assessed_points: 0, earned_points: 0, unassessed_points: 40, ignored_count: 0, criteria: [] },
      findings: [],
    };
  },
  async previewImport(file) {
    if (!file || file.size > 12 * 1024 * 1024) throw new ApiError("Choose a file smaller than 12 MB.", { status: 413, code: "file_too_large" });
    if (!/\.json$/i.test(file.name)) throw new ApiError("The mock API only previews Forma JSON files.", { code: "unsupported_import" });
    const parsed = JSON.parse(await file.text());
    const candidate = parsed?.format === "forma.resume" ? parsed.resume : parsed;
    if (!candidate?.document) throw new ApiError("This does not look like a Forma resume export.", { code: "invalid_import" });
    return { candidate, parser: { id: "forma_json", version: "1" }, mappings: [], warnings: [] };
  },
  async rewriteText({ text, instruction }) {
    if (!mockAIConfig) throw new ApiError("Choose an AI provider first", { status: 409, code: "ai_not_configured" });
    await new Promise((resolve) => setTimeout(resolve, 450));
    const trimmed = text.trim().replace(/\.$/, "");
    return { original: text, proposed: instruction === "shorten" ? `${trimmed.split(" ").slice(0, 12).join(" ")}.` : `${trimmed}.`, rationale: "Tightened wording without adding facts." };
  },
};

const live = {
  health: () => request("/health"),
  listResumes: async () => {
    const result = await request("/resumes");
    return { ...result, items: (result?.items || []).map(fromApiResume) };
  },
  createResume: async (payload) => fromApiResume(await request("/resumes", { method: "POST", body: JSON.stringify(toApiResume(payload)) })),
  getResume: async (id) => fromApiResume(await request(`/resumes/${encodeURIComponent(id)}`)),
  updateResume: async (id, payload) => fromApiResume(await request(`/resumes/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(toApiResume(payload)) })),
  deleteResume: (id) => request(`/resumes/${encodeURIComponent(id)}`, { method: "DELETE" }),
  duplicateResume: async (id) => fromApiResume(await request(`/resumes/${encodeURIComponent(id)}/duplicate`, { method: "POST" })),
  providers: async () => {
    const result = await request("/ai/providers");
    return (result?.items || result?.providers || result || []).map(normalizeProvider);
  },
  getAISession: () => request("/ai/session"),
  setAISession: (payload) => request("/ai/session", { method: "PUT", body: JSON.stringify(payload) }),
  clearAISession: () => request("/ai/session", { method: "DELETE" }),
  evaluateQuality: (payload) => request("/quality/evaluate", { method: "POST", body: JSON.stringify({ resume: toApiDocument(payload.resume) }) }),
  reviewResume: async (payload) => normalizeReview(await request("/ai/review", { method: "POST", body: JSON.stringify({ ...payload, resume: toApiDocument(payload.resume) }) })),
  previewImport: async (file) => {
    const form = new FormData();
    form.append("file", file, file.name);
    const result = await request("/imports/preview", { method: "POST", body: form });
    return { ...result, candidate: result?.candidate?.document ? fromApiResume(result.candidate) : result?.candidate };
  },
  rewriteText: async (payload) => {
    const result = await request("/ai/rewrite", { method: "POST", body: JSON.stringify(payload) });
    return { ...result, proposed: result.rewritten_text, rationale: result.explanation };
  },
};

export const api = USE_MOCK_API ? mock : live;
export { fromApiDocument, normalizeProvider, providerPresets, toApiDocument };
