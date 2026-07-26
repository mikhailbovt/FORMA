import { describe, expect, it } from "vitest";
import { makeSampleResume } from "../data/sampleResume.js";
import { createResumePDFBlob, formatPDFPeriod } from "./pdfExport.jsx";

describe("PDF export", () => {
  it("formats dates with ASCII separators", () => {
    expect(formatPDFPeriod("2022-04", "", true, "en")).toBe("Apr 2022 - Present");
    expect(formatPDFPeriod("2019", "2021", false, "en")).toBe("2019 - 2021");
  });

  it("renders a real PDF blob with a text-capable mixed-script font", async () => {
    const resume = makeSampleResume();
    resume.title = "Mixed script PDF fixture";
    resume.document.basics.full_name = "Alex Morgan";
    resume.document.basics.location = "Moscow";
    resume.document.summary = "Senior Go engineer. \u0421\u043e\u0437\u0434\u0430\u044e \u043d\u0430\u0434\u0435\u0436\u043d\u044b\u0435 \u0441\u0435\u0440\u0432\u0438\u0441\u044b \u043d\u0430 Go \u0438 React.";

    const blob = await createResumePDFBlob(resume);
    const header = new TextDecoder("latin1").decode((await blob.arrayBuffer()).slice(0, 8));

    expect(blob.type).toBe("application/pdf");
    expect(blob.size).toBeGreaterThan(5_000);
    expect(header).toMatch(/^%PDF-/);
  }, 20_000);
});
