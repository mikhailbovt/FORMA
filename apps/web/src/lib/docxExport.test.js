import JSZip from "jszip";
import { describe, expect, it } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { createResumeDOCXBlob, DOCX_MIME, formatDOCXPeriod } from "./docxExport.js";

async function unpack(blob) {
  const zip = await JSZip.loadAsync(await blob.arrayBuffer());
  return {
    zip,
    documentXML: await zip.file("word/document.xml").async("string"),
    relationshipsXML: await zip.file("word/_rels/document.xml.rels").async("string"),
    contentTypesXML: await zip.file("[Content_Types].xml").async("string"),
  };
}

describe("DOCX export", () => {
  it("formats dates with portable ASCII separators", () => {
    expect(formatDOCXPeriod("2022-04", "", true, "en")).toBe("Apr 2022 - Present");
    expect(formatDOCXPeriod("2019", "2021", false, "en")).toBe("2019 - 2021");
  });

  it("creates a real editable OOXML document with ordered, visible sections", async () => {
    const resume = makeSampleResume();
    resume.title = "Editable resume fixture";
    resume.document.section_order = ["basics", "skills", "experience", "private-notes", "projects", "languages"];
    resume.document.hidden_sections = ["projects"];
    resume.document.custom_sections = [{
      id: "private-notes",
      title: "Community",
      items: [{
        id: "community-1",
        title: "Go mentor",
        subtitle: "Open source",
        date: "2025",
        summary: "Mentored first-time contributors.",
        bullets: ["Reviewed 30 pull requests."],
      }],
    }];

    const blob = await createResumeDOCXBlob(resume);
    const bytes = new Uint8Array(await blob.slice(0, 4).arrayBuffer());
    const { zip, documentXML, contentTypesXML } = await unpack(blob);

    expect(blob.type).toBe(DOCX_MIME);
    expect(blob.size).toBeGreaterThan(8_000);
    expect([...bytes]).toEqual([0x50, 0x4b, 0x03, 0x04]);
    expect(zip.file("word/styles.xml")).toBeTruthy();
    expect(zip.file("word/numbering.xml")).toBeTruthy();
    expect(contentTypesXML).toContain("wordprocessingml.document.main+xml");
    expect(documentXML).toContain("Alex Morgan");
    expect(documentXML).toContain("Go mentor");
    expect(documentXML).toContain("Reviewed 30 pull requests.");
    expect(documentXML).not.toContain("Signalboard");
    expect(documentXML.indexOf("Skills")).toBeLessThan(documentXML.indexOf("Experience"));
    expect(documentXML.indexOf("Experience")).toBeLessThan(documentXML.indexOf("Community"));
    expect(documentXML).toContain('w:pgSz w:w="11906" w:h="16838"');
  });

  it("preserves Cyrillic text, hyperlinks, Letter sizing, and a valid optional photo", async () => {
    const resume = makeSampleResume();
    resume.document.paper_size = "letter";
    resume.document.section_order = ["basics", "summary", "certifications", "languages"];
    resume.document.basics.full_name = "Алексей Морозов";
    resume.document.basics.photo_url = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wl2nU0AAAAASUVORK5CYII=";
    resume.document.basics.website = "https://example.com/profile";
    resume.document.basics.profiles.push({ id: "unsafe", network: "Unsafe", url: "javascript:alert(1)" });
    resume.document.summary = "Создаю надёжные сервисы на Go и React.";
    resume.document.certifications = [{ id: "cert-1", name: "Cloud Engineer", issuer: "Acme", date: "2025" }];

    const blob = await createResumeDOCXBlob(resume);
    const { zip, documentXML, relationshipsXML } = await unpack(blob);
    const media = Object.keys(zip.files).filter((name) => name.startsWith("word/media/") && !zip.files[name].dir);

    expect(documentXML).toContain("Алексей Морозов");
    expect(documentXML).toContain("Создаю надёжные сервисы на Go и React.");
    expect(documentXML).toContain("Cloud Engineer");
    expect(documentXML).toContain('w:pgSz w:w="12240" w:h="15840"');
    expect(relationshipsXML).toContain("https://example.com/profile");
    expect(relationshipsXML).not.toContain("javascript:");
    expect(media).toHaveLength(1);
  });
});
