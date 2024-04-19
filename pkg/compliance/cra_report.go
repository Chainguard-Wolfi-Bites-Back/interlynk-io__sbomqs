// Copyright 2024 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"sigs.k8s.io/release-utils/version"
)

var craSectionDetails = map[int]craSection{
	SBOM_SPEC:            {Title: "SBOM formats", Id: "4", Required: true, DataField: "specification"},
	SBOM_SPEC_VERSION:    {Title: "SBOM formats", Id: "4", Required: true, DataField: "specification version"},
	SBOM_BUILD:           {Title: "Level of Detail", Id: "5.1", Required: true, DataField: "build process"},
	SBOM_DEPTH:           {Title: "Level of Detail", Id: "5.1", Required: true, DataField: "depth"},
	SBOM_CREATOR:         {Title: "Required fields sboms ", Id: "5.2.1", Required: true, DataField: "creator of sbom"},
	SBOM_TIMESTAMP:       {Title: "Required fields sboms", Id: "5.2.1", Required: true, DataField: "timestamp"},
	SBOM_COMPONENTS:      {Title: "Required fields component", Id: "5.2.2", Required: true, DataField: "components"},
	SBOM_URI:             {Title: "Additional fields sboms", Id: "5.3.1", Required: false, DataField: "SBOM-URI"},
	COMP_CREATOR:         {Title: "Required fields component", Id: "5.2.2", Required: true, DataField: "component creator"},
	COMP_NAME:            {Title: "Required fields components", Id: "5.2.2", Required: true, DataField: "component name"},
	COMP_VERSION:         {Title: "Required fields components", Id: "5.2.2", Required: true, DataField: "component version"},
	COMP_DEPTH:           {Title: "Required fields components", Id: "5.2.2", Required: true, DataField: "Dependencies on other components"},
	COMP_LICENSE:         {Title: "Required fields components", Id: "5.2.2", Required: true, DataField: "License"},
	COMP_HASH:            {Title: "Required fields components", Id: "5.2.2", Required: true, DataField: "Hash value of the executable component"},
	COMP_SOURCE_CODE_URL: {Title: "Additional fields components", Id: "5.3.2", Required: false, DataField: "Source code URI"},
	COMP_DOWNLOAD_URL:    {Title: "Additional fields components", Id: "5.3.2", Required: false, DataField: "URI of the executable form of the component"},
	COMP_SOURCE_HASH:     {Title: "Additional fields components", Id: "5.3.2", Required: false, DataField: "Hash value of the source code of the component"},
	COMP_OTHER_UNIQ_IDS:  {Title: "Additional fields components", Id: "5.3.2", Required: false, DataField: "Other unique identifiers"},
}

type run struct {
	Id            string `json:"id"`
	GeneratedAt   string `json:"generated_at"`
	FileName      string `json:"file_name"`
	EngineVersion string `json:"compliance_engine_version"`
}
type tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Vendor  string `json:"vendor"`
}
type Summary struct {
	TotalScore         float64 `json:"total_score"`
	MaxScore           float64 `json:"max_score"`
	TotalRequiredScore float64 `json:"required_elements_score"`
	TotalOptionalScore float64 `json:"optional_elements_score"`
}
type craSection struct {
	Title         string  `json:"section_title"`
	Id            string  `json:"section_id"`
	DataField     string  `json:"section_data_field"`
	Required      bool    `json:"required"`
	ElementId     string  `json:"element_id"`
	ElementResult string  `json:"element_result"`
	Score         float64 `json:"score"`
}
type craComplianceReport struct {
	Name     string       `json:"report_name"`
	Subtitle string       `json:"subtitle"`
	Revision string       `json:"revision"`
	Run      run          `json:"run"`
	Tool     tool         `json:"tool"`
	Summary  Summary      `json:"summary"`
	Sections []craSection `json:"sections"`
}

func newJsonReport() *craComplianceReport {
	return &craComplianceReport{
		Name:     "Cyber Resilience Requirements for Manufacturers and Products Report",
		Subtitle: "Part 2: Software Bill of Materials (SBOM)",
		Revision: "TR-03183-2 (1.1)",
		Run: run{
			Id:            uuid.New().String(),
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
			FileName:      "",
			EngineVersion: "1",
		},
		Tool: tool{
			Name:    "sbomqs",
			Version: version.GetVersionInfo().GitVersion,
			Vendor:  "Interlynk (support@interlynk.io)",
		},
	}
}

func craJsonReport(db *db, fileName string) {
	jr := newJsonReport()
	jr.Run.FileName = fileName

	score := craAggregateScore(db)
	summary := Summary{}
	summary.MaxScore = 10.0
	summary.TotalScore = score.totalScore()
	summary.TotalRequiredScore = score.totalRequiredScore()
	summary.TotalOptionalScore = score.totalOptionalScore()

	jr.Summary = summary
	jr.Sections = constructSections(db)

	o, _ := json.MarshalIndent(jr, "", "  ")
	fmt.Println(string(o))
}

func constructSections(db *db) []craSection {
	var sections []craSection
	allIds := db.getAllIds()
	for _, id := range allIds {
		records := db.getRecordsById(id)

		for _, r := range records {
			section := craSectionDetails[r.check_key]
			new_section := craSection{
				Title:     section.Title,
				Id:        section.Id,
				DataField: section.DataField,
				Required:  section.Required,
			}
			score := craKeyIdScore(db, r.check_key, r.id)
			new_section.Score = score.totalScore()
			if r.id == "doc" {
				new_section.ElementId = "sbom"
			} else {
				new_section.ElementId = r.id
			}

			new_section.ElementResult = r.check_value

			sections = append(sections, new_section)
		}
	}
	return sections
}

func craDetailedReport(db *db, fileName string) {
	table := tablewriter.NewWriter(os.Stdout)
	score := craAggregateScore(db)

	fmt.Printf("Cyber Resilience Requirements for Manufacturers and Products Report TR-03183-2 (1.1)\n")
	fmt.Printf("Compliance score by Interlynk Score:%0.1f RequiredScore:%0.1f OptionalScore:%0.1f for %s\n", score.totalScore(), score.totalRequiredScore(), score.totalOptionalScore(), fileName)
	fmt.Printf("* indicates optional fields\n")
	table.SetHeader([]string{"ElementId", "Section", "Datafield", "Element Result", "Score"})
	table.SetRowLine(true)
	table.SetAutoMergeCellsByColumnIndex([]int{0})

	sections := constructSections(db)
	for _, section := range sections {
		sectionId := section.Id
		if !section.Required {
			sectionId = sectionId + "*"
		}
		table.Append([]string{section.ElementId, sectionId, section.DataField, section.ElementResult, fmt.Sprintf("%0.1f", section.Score)})
	}
	table.Render()
}

func craBasicReport(db *db, fileName string) {
	score := craAggregateScore(db)
	fmt.Printf("Cyber Resilience Requirements for Manufacturers and Products Report TR-03183-2 (1.1)\n")
	fmt.Printf("Score:%0.1f RequiredScore:%0.1f OptionalScore:%0.1f for %s\n", score.totalScore(), score.totalRequiredScore(), score.totalOptionalScore(), fileName)
}
