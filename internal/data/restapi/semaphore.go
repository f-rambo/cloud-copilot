package restapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/f-rambo/ocean/internal/conf"

	"github.com/go-resty/resty/v2"
)

var (
	cookiePath  = "/tmp/semaphore-cookie"
	cookibeName = "semaphore-cookie"
)

type Semaphore struct {
	baseurl string
	token   string
	client  *resty.Client
}

func NewSemaphore(c conf.Semaphore) (*Semaphore, error) {
	semaphore := &Semaphore{}
	semaphore.client = resty.New()
	semaphore.baseurl = fmt.Sprintf("http://%s:%d/api/", c.GetHost(), c.GetPort())
	err := semaphore.login(c.GetAdmin(), c.GetAdminPassword())
	if err != nil {
		return nil, err
	}
	err = semaphore.getUserTokens()
	if err != nil {
		return nil, err
	}
	if semaphore.token == "" {
		err = semaphore.createUserToken()
		if err != nil {
			return nil, err
		}
		return semaphore, nil
	}
	return semaphore, nil
}

func (s *Semaphore) login(admin, pass string) error {
	body := map[string]string{"auth": admin, "password": pass}
	bodyByte, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(string(bodyByte)).
		SetOutput(cookiePath).
		Post(s.baseurl + "auth/login")
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("login failed")
	}
	return nil
}

type UserToken struct {
	ID      string `json:"id,omitempty"`
	Created string `json:"created,omitempty"`
	Expired bool   `json:"expired,omitempty"`
	UserID  int    `json:"user_id,omitempty"`
}

func (s *Semaphore) getUserTokens() error {
	usertokens := make([]UserToken, 0)
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetCookies([]*http.Cookie{
			{Name: cookibeName, Value: cookiePath},
		}).SetResult(&usertokens).
		Get(s.baseurl + "user/tokens")

	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("get user token fail , %s", res.String())
	}
	for _, v := range usertokens {
		s.token = v.ID
		break
	}
	return nil
}

func (s *Semaphore) createUserToken() error {
	usertokens := make([]UserToken, 0)
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetCookies([]*http.Cookie{
			{Name: cookibeName, Value: cookiePath},
		}).SetResult(&usertokens).
		Post(s.baseurl + "user/tokens")

	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusCreated {
		return fmt.Errorf("create user token fail , %s", res.String())
	}
	for _, v := range usertokens {
		s.token = v.ID
		break
	}
	return nil
}

// 项目
// {"alert":true,"name":"test2","alert_chat":"111","max_parallel_tasks":1000}
// {"id": 3,"name": "test2","created": "2023-09-05T06:55:09.036124964Z","alert": true,"alert_chat": "111","max_parallel_tasks": 1000}
type Project struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Created     string `json:"created,omitempty"`
	Alert       bool   `json:"alert,omitempty"`
	AlertChat   string `json:"alert_chat,omitempty"`
	MaxParallel int    `json:"max_parallel_tasks,omitempty"`
}

func (s *Semaphore) CreateProject(project *Project) error {
	bodyByte, err := json.Marshal(project)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).SetResult(project).
		Post(fmt.Sprintf("%sprojects", s.baseurl))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusCreated {
		return fmt.Errorf("create project failed, %s", res.String())
	}
	return nil
}

func (s *Semaphore) GetProjects() ([]Project, error) {
	var projects []Project
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetResult(&projects).
		Get(fmt.Sprintf("%sprojects", s.baseurl))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get projects failed, %s", res.String())
	}
	return projects, nil
}

func (s *Semaphore) DeleteProject(projectID int) error {
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Delete(fmt.Sprintf("%sproject/%d", s.baseurl, projectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("delete project failed, %s", res.String())
	}
	return nil
}

// 秘钥key
// {"name": "testKey","type": "login_password","project_id": 1,"login_password": {"password": "password","login": "username"},"ssh": {"login": "user","private_key": "private key"}}
type Key struct {
	ID             int      `json:"id,omitempty"`
	Name           string   `json:"name,omitempty"`
	Type           string   `json:"type,omitempty"`
	ProjectID      int      `json:"project_id,omitempty"`
	LoginPassword  Password `json:"login_password,omitempty"`
	SSH            SSH      `json:"ssh"`
	OverrideSecret bool     `json:"override_secret"`
}

type Password struct {
	Password string `json:"password,omitempty"`
	Login    string `json:"login,omitempty"`
}

type SSH struct {
	Login      string `json:"login,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

func (s *Semaphore) CreateKey(key *Key) error {
	bodyByte, err := json.Marshal(key)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Post(fmt.Sprintf("%sproject/%d/keys", s.baseurl, key.ProjectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("create key failed, %s", res.String())
	}
	keys, err := s.GetKeys(key.ProjectID)
	if err != nil {
		return err
	}
	for _, v := range keys {
		if v.Name == key.Name {
			key.ID = v.ID
			return nil
		}
	}
	return nil
}

func (s *Semaphore) GetKeys(projectID int) ([]Key, error) {
	keys := make([]Key, 0)
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetQueryParams(map[string]string{
			"sort":  "name",
			"order": "asc",
		}).
		SetResult(&keys).
		Get(fmt.Sprintf("%sproject/%d/keys", s.baseurl, projectID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get project failed, %s", res.String())
	}
	return keys, nil
}

func (s *Semaphore) UpdateKey(key Key) error {
	key.OverrideSecret = true
	bodyByte, err := json.Marshal(key)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Put(fmt.Sprintf("%sproject/%d/keys/%d", s.baseurl, key.ProjectID, key.ID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("update project failed, %s", res.String())
	}
	return nil
}

func (s *Semaphore) DeleteKey(projectID, keyID int) error {
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Delete(fmt.Sprintf("%sproject/%d/keys/%d", s.baseurl, projectID, keyID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("delete project failed, %s", res.String())
	}
	return nil
}

// 存储库
// {"name": "Test","project_id": 0,"git_url": "git@example.com","git_branch": "master","ssh_key_id": 0}
type Repository struct {
	ID        int    `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	ProjectID int    `json:"project_id,omitempty"`
	GitURL    string `json:"git_url,omitempty"`
	GitBranch string `json:"git_branch,omitempty"`
	SSHKeyID  int    `json:"ssh_key_id,omitempty"`
}

func (s *Semaphore) CreateRepositories(repo *Repository) error {
	bodyByte, err := json.Marshal(repo)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Post(fmt.Sprintf("%sproject/%d/repositories", s.baseurl, repo.ProjectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("create repositories failed, %s", res.String())
	}
	// 获取repoID
	repos, err := s.GetRepositories(repo.ProjectID)
	if err != nil {
		return err
	}
	for _, v := range repos {
		if v.Name == repo.Name && v.GitBranch == repo.GitBranch && v.GitURL == repo.GitURL {
			repo.ID = v.ID
			break
		}
	}
	return nil
}

func (s *Semaphore) UpdateRepositories(repo *Repository) error {
	bodyByte, err := json.Marshal(repo)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Put(fmt.Sprintf("%sproject/%d/repositories/%d", s.baseurl, repo.ProjectID, repo.ID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("update repositories failed, %s", res.String())
	}
	return nil
}

func (s *Semaphore) GetRepositories(projectID int) ([]Repository, error) {
	var repos []Repository
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetResult(&repos).
		Get(fmt.Sprintf("%sproject/%d/repositories", s.baseurl, projectID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get project failed, %s", res.String())
	}
	return repos, nil
}

func (s *Semaphore) DeleteRepositories(projectID, repoId int) error {
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Delete(fmt.Sprintf("%sproject/%d/repositories/%d", s.baseurl, projectID, repoId))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("delete project failed, %s", res.String())
	}
	return nil
}

// 环境
// {"name": "Test","project_id": 1,"password": "string","json": "{}","env": "{}"}
type Environment struct {
	ID        int    `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	ProjectID int    `json:"project_id,omitempty"`
	Password  string `json:"password,omitempty"`
	JSON      string `json:"json,omitempty"` // json字符串
	Env       string `json:"env,omitempty"`  // json字符串
}

func (s *Semaphore) CreateEnvironment(env *Environment) error {
	if env.JSON == "" {
		env.JSON = "{}"
	}
	if env.Env == "" {
		env.Env = "{}"
	}
	bodyByte, err := json.Marshal(env)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Post(fmt.Sprintf("%sproject/%d/environment", s.baseurl, env.ProjectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("create environment failed, %s", res.String())
	}
	evns, err := s.GetEnvironments(env.ProjectID)
	if err != nil {
		return err
	}
	for _, v := range evns {
		if v.Name == env.Name {
			env.ID = v.ID
			break
		}
	}
	return nil
}

func (s *Semaphore) GetEnvironments(projectID int) ([]Environment, error) {
	var envs []Environment
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetResult(&envs).
		Get(fmt.Sprintf("%sproject/%d/environment", s.baseurl, projectID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get environments failed, %s", res.String())
	}
	return envs, nil
}

func (s *Semaphore) UpdateEnvironments(env Environment) error {
	if env.ID == 0 {
		return fmt.Errorf("inventory id is required")
	}
	if env.ProjectID == 0 {
		return fmt.Errorf("project id is required")
	}
	bodyByte, err := json.Marshal(env)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Put(fmt.Sprintf("%sproject/%d/environment/%d", s.baseurl, env.ProjectID, env.ID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("update environments failed, %s", res.String())
	}
	return nil
}

func (s *Semaphore) DeleteEnvironments(projectID, envID int) error {
	url := fmt.Sprintf("%sproject/%d/environment/%d", s.baseurl, projectID, envID)
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Delete(url)
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("delete environments failed, %s", res.String())
	}
	return nil
}

// 主机配置
type Inventory struct {
	ID        int    `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	ProjectID int    `json:"project_id,omitempty"`
	Inventory string `json:"inventory,omitempty"`
	SSHKeyID  int    `json:"ssh_key_id,omitempty"`
	BecomeKey int    `json:"become_key_id,omitempty"`
	Type      string `json:"type,omitempty"` // Static、Static YAML、File
}

func (s *Semaphore) CreateInventory(inv *Inventory) error {
	// 直接使用struct作为参数就报错
	if inv.ProjectID == 0 {
		return fmt.Errorf("project id is required")
	}
	bodyByte, err := json.Marshal(inv)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).SetResult(inv).
		Post(fmt.Sprintf("%sproject/%d/inventory", s.baseurl, inv.ProjectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusCreated {
		return fmt.Errorf("create inventory failed, %s", res.String())
	}
	return nil
}

func (s *Semaphore) GetInventorys(projectID int) ([]Inventory, error) {
	var invs []Inventory
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetResult(&invs).
		SetPathParams(map[string]string{
			"sort":  "name",
			"order": "asc",
		}).
		Get(fmt.Sprintf("%sproject/%d/inventory", s.baseurl, projectID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get inventorys failed, %s", res.String())
	}
	return invs, nil
}

func (s *Semaphore) UpdateInventory(inv Inventory) error {
	if inv.ID == 0 {
		return fmt.Errorf("inventory id is required")
	}
	if inv.ProjectID == 0 {
		return fmt.Errorf("project id is required")
	}
	bodyByte, err := json.Marshal(inv)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Put(fmt.Sprintf("%sproject/%d/inventory/%d", s.baseurl, inv.ProjectID, inv.ID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("update inventory failed, %s", res.String())
	}
	return nil
}

func (s *Semaphore) DeleteInventory(projectID, invId int) error {
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Delete(fmt.Sprintf("%sproject/%d/inventory/%d", s.baseurl, projectID, invId))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("delete inventory failed, %s", res.String())
	}
	return nil
}

// 任务模版
type Template struct {
	ID                      int         `json:"id,omitempty"`
	ProjectID               int         `json:"project_id,omitempty"`
	Inventory               int         `json:"inventory_id,omitempty"`
	Repository              int         `json:"repository_id,omitempty"`
	Environment             int         `json:"environment_id,omitempty"`
	ViewID                  int         `json:"view_id,omitempty"`
	Name                    string      `json:"name,omitempty"`
	Playbook                string      `json:"playbook,omitempty"`
	Arguments               string      `json:"arguments,omitempty"`
	Description             string      `json:"description,omitempty"`
	Limit                   string      `json:"limit,omitempty"`
	AllowOverrideArgsInTask bool        `json:"allow_override_args_in_task,omitempty"`
	SuppressSuccessAlerts   bool        `json:"suppress_success_alerts,omitempty"`
	LastTask                LastTask    `json:"last_task,omitempty"`
	SurveyVars              []SurveyVar `json:"survey_vars,omitempty"`
}

type LastTask struct {
	ID          int    `json:"id,omitempty"`
	TemplateID  int    `json:"template_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Debug       bool   `json:"debug,omitempty"`
	Playbook    string `json:"playbook,omitempty"`
	Environment string `json:"environment,omitempty"`
	Limit       string `json:"limit,omitempty"`
}

type SurveyVar struct {
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

func (s *Semaphore) CreateTemplate(template *Template) error {
	bodyByte, err := json.Marshal(template)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).SetResult(template).
		Post(fmt.Sprintf("%sproject/%d/templates", s.baseurl, template.ProjectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusCreated {
		return fmt.Errorf("CreateTemplate fail %s", res.String())
	}
	return nil
}

func (s *Semaphore) GetTemplates(projectID int) ([]Template, error) {
	var templates []Template
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetResult(&templates).
		SetPathParams(map[string]string{
			"sort":  "name",
			"order": "asc",
		}).
		Get(fmt.Sprintf("%sproject/%d/templates", s.baseurl, projectID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GetTemplates fail %s", res.String())
	}
	return templates, nil
}

func (s *Semaphore) UpdateTemplate(template Template) error {
	bodyByte, err := json.Marshal(template)
	if err != nil {
		return err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).
		Put(fmt.Sprintf("%sproject/%d/templates/%d", s.baseurl, template.ProjectID, template.ID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("UpdateTemplate fail %s", res.String())
	}
	return nil
}

func (s *Semaphore) DeleteTemplate(projectID, tmID int) error {
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Delete(fmt.Sprintf("%sproject/%d/templates/%d", s.baseurl, projectID, tmID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("DeleteTemplate fail %s", res.String())
	}
	return nil
}

// task
type Task struct {
	ID          int    `json:"id,omitempty"`
	TemplateID  int    `json:"template_id,omitempty"`
	ProjectId   int    `json:"project_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Debug       bool   `json:"debug,omitempty"`
	Diff        bool   `json:"diff,omitempty"`
	Playbook    string `json:"playbook,omitempty"`
	Environment string `json:"environment,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	Limit       string `json:"limit,omitempty"`
}

func (s *Semaphore) GetTasks(projectID int) ([]Task, error) {
	tasks := make([]Task, 0)
	body := map[string]interface{}{"project_id": projectID}
	bodyByte, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).SetResult(&tasks).
		Get(fmt.Sprintf("%sproject/%d/tasks", s.baseurl, projectID))
	if err != nil {
		return nil, err
	}
	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GetTasks fail %s", res.String())
	}
	return tasks, nil
}

// start task
func (s *Semaphore) StartTask(projectID int, task *Task) error {
	bodyByte, err := json.Marshal(task)
	if err != nil {
		return err
	}
	resTask := Task{}
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		SetBody(string(bodyByte)).SetResult(&resTask).
		Post(fmt.Sprintf("%sproject/%d/tasks", s.baseurl, projectID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusCreated {
		return fmt.Errorf("StartTask fail %s", res.String())
	}
	return nil
}

// stop task
func (s *Semaphore) StopTask(projectID, taskID int) error {
	res, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", "Bearer "+s.token).
		Post(fmt.Sprintf("%sproject/%d/tasks/%d/stop", s.baseurl, projectID, taskID))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("StopTask fail %s", res.String())
	}
	return nil
}
