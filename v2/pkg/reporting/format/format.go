package format

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/heckintosh/nuclei/v2/pkg/catalog/config"
	"github.com/heckintosh/nuclei/v2/pkg/utils"

	"github.com/heckintosh/nuclei/v2/pkg/model"
	"github.com/heckintosh/nuclei/v2/pkg/output"
	"github.com/heckintosh/nuclei/v2/pkg/types"
)


// Summary returns a formatted built one line summary of the event
func Summary(event *output.ResultEvent) string {
	template := GetMatchedTemplate(event)

	builder := &strings.Builder{}
	builder.WriteString(types.ToString(event.Info.Name))
	builder.WriteString(" (")
	builder.WriteString(template)
	builder.WriteString(") found on ")
	builder.WriteString(event.Host)
	data := builder.String()
	return data
}

// MarkdownDescription formats a short description of the generated
// event by the nuclei scanner in Markdown format.
func MarkdownDescription(event *output.ResultEvent) string { // TODO remove the code duplication: format.go <-> jira.go
	template := GetMatchedTemplate(event)
	builder := &bytes.Buffer{}
	builder.WriteString("**Details**: **")
	builder.WriteString(template)
	builder.WriteString("** ")

	builder.WriteString(" matched at ")
	builder.WriteString(event.Host)

	builder.WriteString("\n\n**Protocol**: ")
	builder.WriteString(strings.ToUpper(event.Type))

	builder.WriteString("\n\n**Full URL**: ")
	builder.WriteString(event.Matched)

	builder.WriteString("\n\n**Timestamp**: ")
	builder.WriteString(event.Timestamp.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))

	builder.WriteString("\n\n**Template Information**\n\n| Key | Value |\n|---|---|\n")
	builder.WriteString(ToMarkdownTableString(&event.Info))

	if event.Request != "" {
		builder.WriteString(createMarkdownCodeBlock("Request", types.ToHexOrString(event.Request), "http"))
	}
	if event.Response != "" {
		var responseString string
		// If the response is larger than 5 kb, truncate it before writing.
		if len(event.Response) > 5*1024 {
			responseString = (event.Response[:5*1024])
			responseString += ".... Truncated ...."
		} else {
			responseString = event.Response
		}
		builder.WriteString(createMarkdownCodeBlock("Response", responseString, "http"))
	}

	if len(event.ExtractedResults) > 0 || len(event.Metadata) > 0 {
		builder.WriteString("\n**Extra Information**\n\n")

		if len(event.ExtractedResults) > 0 {
			builder.WriteString("**Extracted results**:\n\n")
			for _, v := range event.ExtractedResults {
				builder.WriteString("- ")
				builder.WriteString(v)
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
		if len(event.Metadata) > 0 {
			builder.WriteString("**Metadata**:\n\n")
			for k, v := range event.Metadata {
				builder.WriteString("- ")
				builder.WriteString(k)
				builder.WriteString(": ")
				builder.WriteString(types.ToString(v))
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
	}
	if event.Interaction != nil {
		builder.WriteString("**Interaction Data**\n---\n")
		builder.WriteString(event.Interaction.Protocol)
		if event.Interaction.QType != "" {
			builder.WriteString(" (")
			builder.WriteString(event.Interaction.QType)
			builder.WriteString(")")
		}
		builder.WriteString(" Interaction from ")
		builder.WriteString(event.Interaction.RemoteAddress)
		builder.WriteString(" at ")
		builder.WriteString(event.Interaction.UniqueID)

		if event.Interaction.RawRequest != "" {
			builder.WriteString(createMarkdownCodeBlock("Interaction Request", event.Interaction.RawRequest, ""))
		}
		if event.Interaction.RawResponse != "" {
			builder.WriteString(createMarkdownCodeBlock("Interaction Response", event.Interaction.RawResponse, ""))
		}
	}

	reference := event.Info.Reference
	if !reference.IsEmpty() {
		builder.WriteString("\nReferences: \n")

		referenceSlice := reference.ToSlice()
		for i, item := range referenceSlice {
			builder.WriteString("- ")
			builder.WriteString(item)
			if len(referenceSlice)-1 != i {
				builder.WriteString("\n")
			}
		}
	}
	builder.WriteString("\n")

	if event.CURLCommand != "" {
		builder.WriteString("\n**CURL Command**\n```\n")
		builder.WriteString(types.ToHexOrString(event.CURLCommand))
		builder.WriteString("\n```")
	}

	builder.WriteString(fmt.Sprintf("\n---\nGenerated by [Nuclei %s](https://github.com/heckintosh/nuclei)", config.Version))
	data := builder.String()
	return data
}

// GetMatchedTemplate returns the matched template from a result event
func GetMatchedTemplate(event *output.ResultEvent) string {
	builder := &strings.Builder{}
	builder.WriteString(event.TemplateID)
	if event.MatcherName != "" {
		builder.WriteString(":")
		builder.WriteString(event.MatcherName)
	}
	if event.ExtractorName != "" {
		builder.WriteString(":")
		builder.WriteString(event.ExtractorName)
	}
	template := builder.String()
	return template
}

func ToMarkdownTableString(templateInfo *model.Info) string {
	fields := utils.NewEmptyInsertionOrderedStringMap(5)
	fields.Set("Name", templateInfo.Name)
	fields.Set("Authors", templateInfo.Authors.String())
	fields.Set("Tags", templateInfo.Tags.String())
	fields.Set("Severity", templateInfo.SeverityHolder.Severity.String())
	fields.Set("Description", templateInfo.Description)
	fields.Set("Remediation", templateInfo.Remediation)

	classification := templateInfo.Classification
	if classification != nil {
		if classification.CVSSMetrics != "" {
			generateCVSSMetricsFromClassification(classification, fields)
		}
		generateCVECWEIDLinksFromClassification(classification, fields)
		fields.Set("CVSS-Score", strconv.FormatFloat(classification.CVSSScore, 'f', 2, 64))
	}

	builder := &bytes.Buffer{}

	toMarkDownTable := func(insertionOrderedStringMap *utils.InsertionOrderedStringMap) {
		insertionOrderedStringMap.ForEach(func(key string, value interface{}) {
			switch value := value.(type) {
			case string:
				if utils.IsNotBlank(value) {
					builder.WriteString(fmt.Sprintf("| %s | %s |\n", key, value))
				}
			}
		})
	}

	toMarkDownTable(fields)
	toMarkDownTable(utils.NewInsertionOrderedStringMap(templateInfo.Metadata))

	return builder.String()
}

func generateCVSSMetricsFromClassification(classification *model.Classification, fields *utils.InsertionOrderedStringMap) {
	// Generate cvss link
	var cvssLinkPrefix string
	if strings.Contains(classification.CVSSMetrics, "CVSS:3.0") {
		cvssLinkPrefix = "https://www.first.org/cvss/calculator/3.0#"
	} else if strings.Contains(classification.CVSSMetrics, "CVSS:3.1") {
		cvssLinkPrefix = "https://www.first.org/cvss/calculator/3.1#"
	}
	if cvssLinkPrefix != "" {
		fields.Set("CVSS-Metrics", fmt.Sprintf("[%s](%s%s)", classification.CVSSMetrics, cvssLinkPrefix, classification.CVSSMetrics))
	} else {
		fields.Set("CVSS-Metrics", classification.CVSSMetrics)
	}
}

func generateCVECWEIDLinksFromClassification(classification *model.Classification, fields *utils.InsertionOrderedStringMap) {
	cwes := classification.CWEID.ToSlice()

	cweIDs := make([]string, 0, len(cwes))
	for _, value := range cwes {
		parts := strings.Split(value, "-")
		if len(parts) != 2 {
			continue
		}
		cweIDs = append(cweIDs, fmt.Sprintf("[%s](https://cwe.mitre.org/data/definitions/%s.html)", strings.ToUpper(value), parts[1]))
	}
	if len(cweIDs) > 0 {
		fields.Set("CWE-ID", strings.Join(cweIDs, ","))
	}

	cves := classification.CVEID.ToSlice()

	cveIDs := make([]string, 0, len(cves))
	for _, value := range cves {
		cveIDs = append(cveIDs, fmt.Sprintf("[%s](https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s)", strings.ToUpper(value), value))
	}
	if len(cveIDs) > 0 {
		fields.Set("CVE-ID", strings.Join(cveIDs, ","))
	}
}

func createMarkdownCodeBlock(title string, content string, language string) string {
	return "\n" + createBoldMarkdown(title) + "\n```" + language + "\n" + content + "\n```\n"
}

func createBoldMarkdown(value string) string {
	return "**" + value + "**"
}
