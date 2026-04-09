package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"hawkings-cli/internal/api"
)

func boolToYesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func boolToAvailability(value bool) string {
	if value {
		return "available"
	}
	return "not-available"
}

func intToString(value int) string {
	return fmt.Sprintf("%d", value)
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func mapString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func stringsTrimSpace(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func anyStringOrDash(value any) string {
	switch v := value.(type) {
	case string:
		return valueOrDash(v)
	case int:
		return intToString(v)
	default:
		return "-"
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrOrDash(value *string) string {
	return valueOrDash(stringPtrValue(value))
}

func stringOrNil(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func languageLabel(language *api.Language) string {
	if language == nil {
		return ""
	}
	if language.Code != "" {
		return language.Code
	}
	return language.Name
}

func courseAreaLabel(area *api.CourseAreaSummary) string {
	if area == nil {
		return ""
	}
	if strings.TrimSpace(area.Code) != "" {
		return area.Name + " (" + area.Code + ")"
	}
	return area.Name
}

func joinNames(items []map[string]any) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(mapString(item, "name"))
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func anyInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
	default:
		return 0
	}

	return 0
}

func anyResolvedValue(resolved bool, value any, fallback any) any {
	if resolved {
		return value
	}
	return fallback
}

func anyLen(value any) int {
	switch v := value.(type) {
	case []any:
		return len(v)
	case []map[string]any:
		return len(v)
	default:
		return 0
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func courseSectionModuleCount(section api.CourseSection) int {
	return len(section.CourseModules)
}

func courseAllModuleCount(course api.CourseDetail) int {
	total := len(course.CourseModules)
	for _, section := range course.CourseSections {
		total += len(section.CourseModules)
	}
	return total
}

func courseContentBody(content api.CourseContent) string {
	if content.File == nil {
		return ""
	}
	value, ok := content.File["contents"]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func courseContentFileString(content api.CourseContent, key string) string {
	if content.File == nil {
		return ""
	}
	value, ok := content.File[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func courseContentFileInt(content api.CourseContent, key string) int {
	if content.File == nil {
		return 0
	}
	value, ok := content.File[key]
	if !ok || value == nil {
		return 0
	}
	return anyInt(value)
}

func truncateText(value string, maxChars int) (string, bool, int) {
	totalChars := utf8.RuneCountInString(value)
	if maxChars <= 0 || totalChars <= maxChars {
		return value, false, totalChars
	}

	runes := []rune(value)
	return string(runes[:maxChars]), true, totalChars
}

func programHasSyllabus(program api.ProgramDetail) bool {
	return program.Syllabus != nil
}

func programHasCourses(program api.ProgramDetail) bool {
	return len(program.Courses) > 0 || anyInt(program.CoursesCount) > 0
}

func canonicalProgramStatus(status string) string {
	status = strings.TrimSpace(status)
	switch status {
	case "courses_created":
		return "courses-created"
	default:
		return status
	}
}

func normalizeProgramSummary(program api.ProgramSummary) api.ProgramSummary {
	status := canonicalProgramStatus(stringPtrValue(program.Status))
	if status == "" {
		program.Status = nil
		return program
	}
	program.Status = ptr(status)
	return program
}

func normalizeProgramList(list api.ProgramList) api.ProgramList {
	for i := range list.Data {
		list.Data[i] = normalizeProgramSummary(list.Data[i])
	}
	return list
}

func normalizeProgramDetail(program api.ProgramDetail) api.ProgramDetail {
	status := canonicalProgramStatus(stringPtrValue(program.Status))
	if status == "" {
		program.Status = nil
		return program
	}
	program.Status = ptr(status)
	return program
}

func programStatusHint(program api.ProgramDetail) string {
	status := canonicalProgramStatus(stringPtrValue(program.Status))
	hasSyllabus := programHasSyllabus(program)
	hasCourses := programHasCourses(program)

	switch status {
	case "created":
		if hasSyllabus {
			return "El programa sigue en created pero ya tiene syllabus; todavia no hay cursos creados."
		}
		return "Programa en fase inicial; normalmente aun no tiene syllabus final ni cursos."
	case "syllabus-processing":
		return "El syllabus se esta generando; la estructura puede ser parcial."
	case "syllabus-processed", "structure-generated":
		return "El syllabus esta listo; todavia no deberia haber cursos."
	case "courses-creating":
		return "La creacion de cursos esta en curso; el arbol puede estar incompleto."
	case "courses-created", "courses_created":
		return "Los cursos ya existen; se pueden navegar sections y modules."
	case "processing", "processed", "completed", "content-added":
		return "Programa en fase avanzada; prioriza navegar por courses y modules reales."
	case "syllabus-error", "error":
		return "Programa con error; revisa config y syllabus antes de asumir estructura."
	case "":
		if hasCourses {
			return "Estado nulo pero hay cursos; navega por presencia real de datos."
		}
		if hasSyllabus {
			return "Estado nulo pero hay syllabus; navega por syllabus antes de courses."
		}
		return "Estado nulo y sin estructura expandida; trata el programa como legacy o incompleto."
	default:
		if hasCourses {
			return "Estado no tipado pero hay cursos; navega por el arbol real."
		}
		if hasSyllabus {
			return "Estado no tipado pero hay syllabus; aun no hay cursos."
		}
		return "Estado no tipado; comprueba syllabus y courses por presencia real."
	}
}

func normalizedStatus(value *string) string {
	status := canonicalProgramStatus(stringPtrValue(value))
	if status == "" {
		return "unknown"
	}
	return status
}

func programStatusSortKey(status string) int {
	status = canonicalProgramStatus(status)
	order := []string{
		"created",
		"syllabus-processing",
		"syllabus-processed",
		"courses-creating",
		"courses-created",
		"processing",
		"processed",
		"completed",
		"content-added",
		"structure-generated",
		"syllabus-error",
		"error",
		"unknown",
	}
	index := slices.Index(order, status)
	if index >= 0 {
		return index
	}
	return len(order) + 1
}

func printOptionalBlock(title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	writeLine("")
	writeLine("%s:", title)
	writeLine("%s", body)
}

func ptr[T any](value T) *T {
	return &value
}
