import { describe, expect, it } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { fromApiDocument, normalizeProvider, toApiDocument } from "./api.js";

describe("API contract adapter", () => {
  it("serializes the editor document into the strict Go schema", () => {
    const source = makeSampleResume().document;
    source.hidden_sections = ["projects"];
    source.basics.photo_url = "data:image/jpeg;base64,cGhvdG8=";
    const document = toApiDocument(source);

    expect(document.basics.name).toBe("Alex Morgan");
    expect(document.basics).not.toHaveProperty("full_name");
    expect(document.basics.photo_url).toBe(source.basics.photo_url);
    expect(document.order).toEqual(source.section_order);
    expect(document.page_size).toBe("A4");
    expect(document.projects[0].summary).toBe(source.projects[0].description);
    expect(document.skills[0].keywords).toEqual(source.skills[0].items);
    expect(document.hidden_sections).toEqual(["projects"]);
  });

  it("deserializes canonical API fields back into editor fields", () => {
    const source = makeSampleResume().document;
    source.basics.photo_url = "data:image/jpeg;base64,cGhvdG8=";
    const document = fromApiDocument(toApiDocument(source));

    expect(document.basics.full_name).toBe("Alex Morgan");
    expect(document.basics.photo_url).toBe(source.basics.photo_url);
    expect(document.section_order).toEqual(source.section_order);
    expect(document.paper_size).toBe("a4");
    expect(document.education[0].degree).toBe(source.education[0].degree);
    expect(document.languages[0].language).toBe("English");
  });

  it("normalizes the backend provider catalog for the settings dialog", () => {
    const provider = normalizeProvider({
      id: "ollama",
      default_model: "qwen3",
      default_base_url: "http://host.docker.internal:11434/v1",
      requires_api_key: false,
      supports_custom_base_url: true,
    });

    expect(provider.suggested_model).toBe("qwen3");
    expect(provider.base_url).toContain("11434");
    expect(provider.key_required).toBe(false);
    expect(provider.base_url_editable).toBe(true);
  });
});
