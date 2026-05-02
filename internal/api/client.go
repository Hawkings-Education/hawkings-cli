package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hawkings-cli/internal/config"
)

const userAgent = "hawkings/0.1.0"
const maxResponseBytes = 16 << 20

type Client struct {
	baseURL string
	xAPIKey string
	client  *http.Client
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("hawkings API request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("hawkings API request failed with status %d: %s", e.StatusCode, e.Body)
}

type Language struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
	RTL  bool   `json:"rtl"`
}

type LearningPlatform struct {
	ID   int     `json:"id"`
	UUID string  `json:"uuid"`
	Name string  `json:"name"`
	Logo *string `json:"logo"`
}

type Profile struct {
	ID               int               `json:"id"`
	Name             string            `json:"name"`
	Surname          string            `json:"surname"`
	Email            string            `json:"email"`
	Admin            bool              `json:"admin"`
	Manager          bool              `json:"manager"`
	Teacher          bool              `json:"teacher"`
	Student          bool              `json:"student"`
	Language         *Language         `json:"language"`
	LearningPlatform *LearningPlatform `json:"learning_platform"`
}

type UserSummary struct {
	ID      any    `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Email   string `json:"email,omitempty"`
}

type CourseContent struct {
	ID              int            `json:"id"`
	Name            string         `json:"name"`
	Type            string         `json:"type"`
	Mime            string         `json:"mime"`
	Status          string         `json:"status"`
	URL             string         `json:"url"`
	Enabled         bool           `json:"enabled"`
	RemoteID        *string        `json:"remote_id"`
	RemoteUpdatedAt *string        `json:"remote_updated_at"`
	File            map[string]any `json:"file"`
}

type Activity struct {
	ID            int              `json:"id"`
	UUID          string           `json:"uuid"`
	Type          string           `json:"type"`
	Title         string           `json:"title"`
	Status        string           `json:"status"`
	Description   string           `json:"description"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
	Input         any              `json:"input"`
	Content       any              `json:"content"`
	Questions     []map[string]any `json:"questions"`
	CourseModules []map[string]any `json:"course_modules"`
}

type CourseModule struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	URL            string          `json:"url"`
	Order          int             `json:"order"`
	RemoteID       *string         `json:"remote_id"`
	Metadata       map[string]any  `json:"metadata"`
	Status         *string         `json:"status"`
	Enabled        bool            `json:"enabled"`
	ApprovedAt     *string         `json:"approved_at"`
	CourseContents []CourseContent `json:"course_contents"`
	Activity       *Activity       `json:"activity"`
}

type CourseSection struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	Order         int            `json:"order"`
	RemoteID      *string        `json:"remote_id"`
	Metadata      map[string]any `json:"metadata"`
	Enabled       bool           `json:"enabled"`
	Course        map[string]any `json:"course"`
	CourseModules []CourseModule `json:"course_modules"`
}

type CourseDetail struct {
	ID                       int             `json:"id"`
	Name                     string          `json:"name"`
	AIBehaviour              *string         `json:"ai_behaviour"`
	GradePublishDelay        any             `json:"grade_publish_delay"`
	Status                   *string         `json:"status"`
	RemoteID                 *string         `json:"remote_id"`
	Metadata                 map[string]any  `json:"metadata"`
	Image                    *string         `json:"image"`
	Enabled                  bool            `json:"enabled"`
	CreatedAt                string          `json:"created_at"`
	UpdatedAt                string          `json:"updated_at"`
	CourseContentProcessedAt *string         `json:"course_content_processed_at"`
	PromptEvaluatorModel     *string         `json:"prompt_evaluator_model"`
	AssignmentsCount         any             `json:"assignments_count"`
	CourseModules            []CourseModule  `json:"course_modules"`
	CourseSections           []CourseSection `json:"course_sections"`
	Language                 *Language       `json:"language"`
}

type ProgramSummary struct {
	ID                       int              `json:"id"`
	Name                     string           `json:"name"`
	RemoteID                 *string          `json:"remote_id"`
	Enabled                  bool             `json:"enabled"`
	Status                   *string          `json:"status"`
	Syllabus                 any              `json:"syllabus"`
	SyllabusPrompt           *string          `json:"syllabus_prompt"`
	CourseModulePromptCustom *string          `json:"course_module_prompt_custom"`
	ResearchInstructions     *string          `json:"research_instructions"`
	Context                  *string          `json:"context"`
	CreatedAt                string           `json:"created_at"`
	UpdatedAt                string           `json:"updated_at"`
	Metadata                 map[string]any   `json:"metadata"`
	CoursesCount             any              `json:"courses_count"`
	Language                 *Language        `json:"language"`
	User                     *UserSummary     `json:"user"`
	Spaces                   []map[string]any `json:"spaces"`
}

type ProgramDetail struct {
	ID                       int              `json:"id"`
	Name                     string           `json:"name"`
	Image                    *string          `json:"image"`
	Syllabus                 any              `json:"syllabus"`
	SyllabusPrompt           *string          `json:"syllabus_prompt"`
	CourseModulePromptCustom *string          `json:"course_module_prompt_custom"`
	ResearchInstructions     *string          `json:"research_instructions"`
	Context                  *string          `json:"context"`
	RemoteID                 *string          `json:"remote_id"`
	Enabled                  bool             `json:"enabled"`
	Status                   *string          `json:"status"`
	CreatedAt                string           `json:"created_at"`
	UpdatedAt                string           `json:"updated_at"`
	CourseFaculty            map[string]any   `json:"course_faculty"`
	CourseProgramTemplate    map[string]any   `json:"course_program_template"`
	Courses                  []CourseDetail   `json:"courses"`
	CoursesCount             any              `json:"courses_count"`
	Language                 *Language        `json:"language"`
	Spaces                   []map[string]any `json:"spaces"`
	SpacesCount              any              `json:"spaces_count"`
	User                     *UserSummary     `json:"user"`
	Users                    []UserSummary    `json:"users"`
	UsersCount               any              `json:"users_count"`
	Metadata                 map[string]any   `json:"metadata"`
}

type ProgramList struct {
	Data   []ProgramSummary `json:"data"`
	Pages  int              `json:"pages"`
	Page   int              `json:"page"`
	Offset int              `json:"offset"`
	Total  int              `json:"total"`
}

type CourseSummary struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Status    *string        `json:"status"`
	RemoteID  *string        `json:"remote_id"`
	Image     *string        `json:"image"`
	Enabled   bool           `json:"enabled"`
	CreatedAt string         `json:"created_at"`
	Metadata  map[string]any `json:"metadata"`
	Language  *Language      `json:"language"`
}

type CourseList struct {
	Data   []CourseSummary `json:"data"`
	Pages  int             `json:"pages"`
	Page   int             `json:"page"`
	Offset int             `json:"offset"`
	Total  int             `json:"total"`
}

type CourseAreaSummary struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type CourseFaculty struct {
	ID          int                `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Code        string             `json:"code"`
	Enabled     bool               `json:"enabled"`
	CourseArea  *CourseAreaSummary `json:"course_area"`
	User        *UserSummary       `json:"user"`
}

type CourseFacultyList struct {
	Data   []CourseFaculty `json:"data"`
	Pages  int             `json:"pages"`
	Page   int             `json:"page"`
	Offset int             `json:"offset"`
	Total  int             `json:"total"`
}

type CourseProgramTemplateModule struct {
	ID          int     `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type CourseProgramTemplateRelation struct {
	ID       int                          `json:"id"`
	Scope    string                       `json:"scope"`
	Position string                       `json:"position"`
	Duration any                          `json:"duration"`
	Order    int                          `json:"order"`
	Metadata map[string]any               `json:"metadata"`
	Module   *CourseProgramTemplateModule `json:"module"`
}

type CourseProgramTemplate struct {
	ID              int                             `json:"id"`
	Code            string                          `json:"code"`
	Name            string                          `json:"name"`
	Description     *string                         `json:"description"`
	CoursesMin      any                             `json:"courses_min"`
	CoursesMax      any                             `json:"courses_max"`
	CoursesHoursMin any                             `json:"courses_hours_min"`
	CoursesHoursMax any                             `json:"courses_hours_max"`
	Metadata        map[string]any                  `json:"metadata"`
	Related         []CourseProgramTemplateRelation `json:"related"`
}

type SpaceSummary struct {
	ID                  int           `json:"id"`
	Name                string        `json:"name"`
	RemoteID            *string       `json:"remote_id"`
	Description         *string       `json:"description"`
	Color               string        `json:"color"`
	Personal            bool          `json:"personal"`
	Enabled             bool          `json:"enabled"`
	CreatedAt           string        `json:"created_at"`
	UpdatedAt           string        `json:"updated_at"`
	CourseProgramsCount any           `json:"course_programs_count"`
	User                *UserSummary  `json:"user"`
	Users               []UserSummary `json:"users"`
	UsersCount          any           `json:"users_count"`
}

type SpaceDetail struct {
	ID                  int             `json:"id"`
	Name                string          `json:"name"`
	RemoteID            *string         `json:"remote_id"`
	Description         *string         `json:"description"`
	Color               string          `json:"color"`
	Personal            bool            `json:"personal"`
	Enabled             bool            `json:"enabled"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
	CoursePrograms      []ProgramDetail `json:"course_programs"`
	CourseProgramsCount any             `json:"course_programs_count"`
	User                *UserSummary    `json:"user"`
	Users               []UserSummary   `json:"users"`
	UsersCount          any             `json:"users_count"`
}

type SpaceList struct {
	Data   []SpaceSummary `json:"data"`
	Pages  int            `json:"pages"`
	Page   int            `json:"page"`
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
}

func NewClient(cfg config.ResolvedConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		xAPIKey: cfg.XAPIKey,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) GetProfile(ctx context.Context) (Profile, error) {
	var profile Profile
	if err := c.getJSON(ctx, "/profile", nil, &profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (c *Client) GetPlatforms(ctx context.Context) ([]LearningPlatform, error) {
	var platforms []LearningPlatform
	if err := c.getJSON(ctx, "/profile/learning-platform", nil, &platforms); err != nil {
		return nil, err
	}
	return platforms, nil
}

func (c *Client) ListLanguages(ctx context.Context) ([]Language, error) {
	var languages []Language
	if err := c.getJSON(ctx, "/language", nil, &languages); err != nil {
		return nil, err
	}
	return languages, nil
}

func (c *Client) ListCourseFaculties(ctx context.Context, params url.Values, with []string) (CourseFacultyList, error) {
	var list CourseFacultyList
	params = withValues(params, with)
	if err := c.getJSON(ctx, "/course-faculty", params, &list); err != nil {
		return CourseFacultyList{}, err
	}
	return list, nil
}

func (c *Client) ListProgramTemplates(ctx context.Context) ([]CourseProgramTemplate, error) {
	var templates []CourseProgramTemplate
	if err := c.getJSON(ctx, "/course-program-template", nil, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (c *Client) ListPrograms(ctx context.Context, params url.Values) (ProgramList, error) {
	var list ProgramList
	if err := c.getJSON(ctx, "/course-program", params, &list); err != nil {
		return ProgramList{}, err
	}
	return list, nil
}

func (c *Client) ListCourses(ctx context.Context, params url.Values, with []string) (CourseList, error) {
	var list CourseList
	params = withValues(params, with)
	if err := c.getJSON(ctx, "/course", params, &list); err != nil {
		return CourseList{}, err
	}
	return list, nil
}

func (c *Client) ListSpaces(ctx context.Context, params url.Values, with []string) (SpaceList, error) {
	var list SpaceList
	params = withValues(params, with)
	if err := c.getJSON(ctx, "/space", params, &list); err != nil {
		return SpaceList{}, err
	}
	return list, nil
}

func (c *Client) GetSpace(ctx context.Context, id string, with []string) (SpaceDetail, error) {
	var space SpaceDetail
	params := withValues(nil, with)
	if err := c.getJSON(ctx, "/space/"+id, params, &space); err != nil {
		return SpaceDetail{}, err
	}
	return space, nil
}

func (c *Client) GetProgram(ctx context.Context, id string, with []string) (ProgramDetail, error) {
	var program ProgramDetail
	params := withValues(nil, with)
	if err := c.getJSON(ctx, "/course-program/"+id, params, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) GetCourse(ctx context.Context, id string, with []string) (CourseDetail, error) {
	var course CourseDetail
	params := withValues(nil, with)
	if err := c.getJSON(ctx, "/course/"+id, params, &course); err != nil {
		return CourseDetail{}, err
	}
	return course, nil
}

func (c *Client) GetCourseModule(ctx context.Context, id string, with []string, contents bool) (CourseModule, error) {
	var module CourseModule
	params := withValues(nil, with)
	if contents {
		params.Set("contents", "true")
	}
	if err := c.getJSON(ctx, "/course-module/"+id, params, &module); err != nil {
		return CourseModule{}, err
	}
	return module, nil
}

func (c *Client) GetCourseSection(ctx context.Context, id string, with []string) (CourseSection, error) {
	var section CourseSection
	params := withValues(nil, with)
	if err := c.getJSON(ctx, "/course-section/"+id, params, &section); err != nil {
		return CourseSection{}, err
	}
	return section, nil
}

func (c *Client) GetCourseContent(ctx context.Context, id string, contents bool) (CourseContent, error) {
	var content CourseContent
	params := url.Values{}
	if contents {
		params.Set("contents", "true")
	}
	if err := c.getJSON(ctx, "/course-content/"+id, params, &content); err != nil {
		return CourseContent{}, err
	}
	return content, nil
}

func (c *Client) GetActivity(ctx context.Context, id string, with []string) (Activity, error) {
	var activity Activity
	params := withValues(nil, with)
	if err := c.getJSON(ctx, "/activity/"+id, params, &activity); err != nil {
		return Activity{}, err
	}
	return activity, nil
}

func (c *Client) GetCourseModulesStatus(ctx context.Context, courseID string) (map[string]string, error) {
	var statuses map[string]string
	if err := c.getJSON(ctx, "/course/"+courseID+"/course-module/status", nil, &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (c *Client) CreateCourseBulk(ctx context.Context, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	if err := c.sendJSON(ctx, http.MethodPost, "/course/bulk", nil, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UploadCourseImage(ctx context.Context, id string, fields map[string]any, imagePath string) (CourseDetail, error) {
	var course CourseDetail
	if err := c.sendMultipart(ctx, http.MethodPatch, "/course/"+id, fields, "image", imagePath, &course); err != nil {
		return CourseDetail{}, err
	}
	return course, nil
}

func (c *Client) GenerateCourseImage(ctx context.Context, id string, force bool) (CourseDetail, error) {
	var course CourseDetail
	payload := map[string]any{"force": force}
	if err := c.sendJSON(ctx, http.MethodPost, "/course/"+id+"/image/generate", nil, payload, &course); err != nil {
		return CourseDetail{}, err
	}
	return course, nil
}

func (c *Client) CreateScorm(ctx context.Context, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	if err := c.sendJSON(ctx, http.MethodPost, "/scorm", nil, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateCourseModuleOnly(ctx context.Context, id string, payload map[string]any) (CourseModule, error) {
	var module CourseModule
	if err := c.sendJSON(ctx, http.MethodPatch, "/course-module/"+id+"/only", nil, payload, &module); err != nil {
		return CourseModule{}, err
	}
	return module, nil
}

func (c *Client) CreateCourseModule(ctx context.Context, payload map[string]any) (CourseModule, error) {
	var module CourseModule
	if err := c.sendJSON(ctx, http.MethodPost, "/course-module", nil, payload, &module); err != nil {
		return CourseModule{}, err
	}
	return module, nil
}

func (c *Client) CreateCourseContent(ctx context.Context, payload map[string]any) (CourseContent, error) {
	var content CourseContent
	if err := c.sendJSON(ctx, http.MethodPost, "/course-content", nil, payload, &content); err != nil {
		return CourseContent{}, err
	}
	return content, nil
}

func (c *Client) UpdateCourseContent(ctx context.Context, id string, payload map[string]any) (CourseContent, error) {
	var content CourseContent
	if err := c.sendJSON(ctx, http.MethodPatch, "/course-content/"+id, nil, payload, &content); err != nil {
		return CourseContent{}, err
	}
	return content, nil
}

func (c *Client) UpdateActivity(ctx context.Context, id string, payload map[string]any) (Activity, error) {
	var activity Activity
	if err := c.sendJSON(ctx, http.MethodPatch, "/activity/"+id, nil, payload, &activity); err != nil {
		return Activity{}, err
	}
	return activity, nil
}

func (c *Client) DeleteCourseContent(ctx context.Context, id string) error {
	return c.sendJSON(ctx, http.MethodDelete, "/course-content/"+id, nil, nil, nil)
}

func (c *Client) GenerateCourseModuleContent(ctx context.Context, id string, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	if err := c.sendJSON(ctx, http.MethodPost, "/course-module/"+id+"/course-content/generate", nil, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GenerateCourseModuleActivity(ctx context.Context, id string, payload map[string]any) (CourseModule, error) {
	var module CourseModule
	if err := c.sendJSON(ctx, http.MethodPost, "/course-module/"+id+"/activity/generate", nil, payload, &module); err != nil {
		return CourseModule{}, err
	}
	return module, nil
}

func (c *Client) ApproveCourseModule(ctx context.Context, id string, approved bool) (CourseModule, error) {
	var module CourseModule
	payload := map[string]any{"approved_at": nil}
	if approved {
		payload["approved_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	if err := c.sendJSON(ctx, http.MethodPatch, "/course-module/"+id+"/boolean/approved_at", nil, payload, &module); err != nil {
		return CourseModule{}, err
	}
	return module, nil
}

func (c *Client) GenerateCourseSectionContent(ctx context.Context, id string, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	if err := c.sendJSON(ctx, http.MethodPost, "/course-section/"+id+"/course-content/generate", nil, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GenerateCourseSectionActivities(ctx context.Context, id string, payload map[string]any) (map[string]any, error) {
	var result map[string]any
	if err := c.sendJSON(ctx, http.MethodPost, "/course-section/"+id+"/activity/generate", nil, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateProgram(ctx context.Context, payload map[string]any) (ProgramDetail, error) {
	var program ProgramDetail
	if err := c.sendJSON(ctx, http.MethodPost, "/course-program", nil, payload, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) UpdateProgramOnly(ctx context.Context, id string, payload map[string]any) (ProgramDetail, error) {
	var program ProgramDetail
	if err := c.sendJSON(ctx, http.MethodPatch, "/course-program/"+id+"/only", nil, payload, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) UploadProgramImage(ctx context.Context, id string, fields map[string]any, imagePath string) (ProgramDetail, error) {
	var program ProgramDetail
	if err := c.sendMultipart(ctx, http.MethodPatch, "/course-program/"+id, fields, "image", imagePath, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) GenerateProgramImage(ctx context.Context, id string, force bool) (ProgramDetail, error) {
	var program ProgramDetail
	payload := map[string]any{"force": force}
	if err := c.sendJSON(ctx, http.MethodPost, "/course-program/"+id+"/image/generate", nil, payload, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) UpdateProgramSpaces(ctx context.Context, id string, selected []int) ([]map[string]any, error) {
	var spaces []map[string]any
	payload := map[string]any{"selected": selected}
	if err := c.sendJSON(ctx, http.MethodPost, "/course-program/"+id+"/space", nil, payload, &spaces); err != nil {
		return nil, err
	}
	return spaces, nil
}

func (c *Client) GenerateProgramSyllabus(ctx context.Context, id string, payload map[string]any) (ProgramDetail, error) {
	var program ProgramDetail
	if err := c.sendJSON(ctx, http.MethodPost, "/course-program/"+id+"/syllabus/generate", nil, payload, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) CreateProgramCoursesFromSyllabus(ctx context.Context, id string, payload map[string]any) (ProgramDetail, error) {
	var program ProgramDetail
	if payload == nil {
		payload = map[string]any{}
	}
	if err := c.sendJSON(ctx, http.MethodPost, "/course-program/"+id+"/syllabus/course", nil, payload, &program); err != nil {
		return ProgramDetail{}, err
	}
	return program, nil
}

func (c *Client) UpdateProgramCourses(ctx context.Context, id string, payload map[string]any) ([]CourseDetail, error) {
	var courses []CourseDetail
	if err := c.sendJSON(ctx, http.MethodPost, "/course-program/"+id+"/course", nil, payload, &courses); err != nil {
		return nil, err
	}
	return courses, nil
}

func (c *Client) DeleteProgram(ctx context.Context, id string) error {
	return c.sendJSON(ctx, http.MethodDelete, "/course-program/"+id, nil, nil, nil)
}

func (c *Client) CreateSpace(ctx context.Context, payload map[string]any) (SpaceDetail, error) {
	var space SpaceDetail
	if err := c.sendJSON(ctx, http.MethodPost, "/space", nil, payload, &space); err != nil {
		return SpaceDetail{}, err
	}
	return space, nil
}

func (c *Client) UpdateSpace(ctx context.Context, id string, payload map[string]any) (SpaceDetail, error) {
	var space SpaceDetail
	if err := c.sendJSON(ctx, http.MethodPatch, "/space/"+id, nil, payload, &space); err != nil {
		return SpaceDetail{}, err
	}
	return space, nil
}

func (c *Client) GetSpacePrograms(ctx context.Context, id string, with []string) ([]ProgramDetail, error) {
	var programs []ProgramDetail
	params := withValues(nil, with)
	if err := c.getJSON(ctx, "/space/"+id+"/course-program", params, &programs); err != nil {
		return nil, err
	}
	return programs, nil
}

func (c *Client) UpdateSpacePrograms(ctx context.Context, id string, payload map[string]any) ([]ProgramDetail, error) {
	var programs []ProgramDetail
	if err := c.sendJSON(ctx, http.MethodPatch, "/space/"+id+"/course-program", nil, payload, &programs); err != nil {
		return nil, err
	}
	return programs, nil
}

func (c *Client) GetSpaceUsers(ctx context.Context, id string) ([]UserSummary, error) {
	var users []UserSummary
	if err := c.getJSON(ctx, "/space/"+id+"/user", nil, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (c *Client) UpdateSpaceUsers(ctx context.Context, id string, payload map[string]any) ([]UserSummary, error) {
	var users []UserSummary
	if err := c.sendJSON(ctx, http.MethodPatch, "/space/"+id+"/user", nil, payload, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (c *Client) DeleteSpace(ctx context.Context, id string) error {
	return c.sendJSON(ctx, http.MethodDelete, "/space/"+id, nil, nil, nil)
}

func (c *Client) GenerateImage(ctx context.Context, payload map[string]any) (string, error) {
	return c.sendForString(ctx, http.MethodPost, "/prompt/tool/image", nil, payload)
}

func (c *Client) sendForString(ctx context.Context, method, endpoint string, params url.Values, payload any) (string, error) {
	urlStr := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		urlStr += "?" + params.Encode()
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("x-api-key", c.xAPIKey)

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(data)),
		}
	}

	return strings.TrimSpace(string(data)), nil
}

func (c *Client) sendMultipart(ctx context.Context, method, endpoint string, fields map[string]any, fileField string, filePath string, out any) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range fields {
		if value == nil {
			continue
		}
		text, err := multipartFieldValue(value)
		if err != nil {
			return fmt.Errorf("encode multipart field %s: %w", key, err)
		}
		if err := writer.WriteField(key, text); err != nil {
			return fmt.Errorf("write multipart field %s: %w", key, err)
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create multipart image field: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy image file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, &body)
	if err != nil {
		return fmt.Errorf("build multipart request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("x-api-key", c.xAPIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute multipart request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		if err != nil {
			return fmt.Errorf("read multipart error response: %w", err)
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(data)),
		}
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("decode multipart response: %w", err)
	}
	return nil
}

func multipartFieldValue(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", typed), nil
	case int32:
		return fmt.Sprintf("%d", typed), nil
	case int64:
		return fmt.Sprintf("%d", typed), nil
	case float32:
		return fmt.Sprintf("%v", typed), nil
	case float64:
		return fmt.Sprintf("%v", typed), nil
	case json.Number:
		return typed.String(), nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func (c *Client) getJSON(ctx context.Context, endpoint string, params url.Values, out any) error {
	return c.sendJSON(ctx, http.MethodGet, endpoint, params, nil, out)
}

func (c *Client) sendJSON(ctx context.Context, method, endpoint string, params url.Values, payload any, out any) error {
	urlStr := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		urlStr += "?" + params.Encode()
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("x-api-key", c.xAPIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		if err != nil {
			return fmt.Errorf("read error response: %w", err)
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

func withValues(values url.Values, with []string) url.Values {
	if values == nil {
		values = url.Values{}
	}
	for _, item := range with {
		item = strings.TrimSpace(item)
		if item != "" {
			values.Add("with[]", item)
		}
	}
	return values
}
