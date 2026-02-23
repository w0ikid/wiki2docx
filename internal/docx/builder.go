package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafeChars = regexp.MustCompile(`[\\/:*?"<>| ]+`)

// safeFilename converts an article title into a safe filename.
func safeFilename(title string) string {
	s := unsafeChars.ReplaceAllString(title, "_")
	s = strings.Trim(s, "_")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// Build creates a .docx file for the given article and saves it to outDir.
func Build(title, content, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Split on double newlines for paragraph breaks; single newlines become spaces.
	chunks := strings.Split(content, "\n\n")
	bodyParagraphs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		// Collapse internal single newlines into a space.
		chunk = strings.ReplaceAll(chunk, "\n", " ")
		bodyParagraphs = append(bodyParagraphs, chunk)
	}

	filename := safeFilename(title) + ".docx"
	outPath := filepath.Join(outDir, filename)
	if err := writeDocx(outPath, title, bodyParagraphs); err != nil {
		return fmt.Errorf("save docx: %w", err)
	}
	return nil
}

func writeDocx(path, title string, paragraphs []string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	addFile := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(content))
		return err
	}

	if err := addFile("[Content_Types].xml", contentTypesXML); err != nil {
		return fmt.Errorf("write [Content_Types].xml: %w", err)
	}
	if err := addFile("_rels/.rels", rootRelsXML); err != nil {
		return fmt.Errorf("write _rels/.rels: %w", err)
	}
	if err := addFile("word/document.xml", buildDocumentXML(title, paragraphs)); err != nil {
		return fmt.Errorf("write word/document.xml: %w", err)
	}

	return nil
}

func buildDocumentXML(title string, paragraphs []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)
	b.WriteString(`<w:p><w:r><w:rPr><w:b/><w:sz w:val="48"/></w:rPr><w:t xml:space="preserve">`)
	b.WriteString(xmlEscape(title))
	b.WriteString(`</w:t></w:r></w:p>`)

	for _, p := range paragraphs {
		b.WriteString(`<w:p><w:pPr><w:jc w:val="both"/></w:pPr><w:r><w:rPr><w:sz w:val="24"/></w:rPr><w:t xml:space="preserve">`)
		b.WriteString(xmlEscape(p))
		b.WriteString(`</w:t></w:r></w:p>`)
	}

	b.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return ""
	}
	return buf.String()
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
