package biz

type Service struct {
	ID         int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name       string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Repo       string `json:"repo,omitempty" gorm:"column:repo; default:''; NOT NULL"`
	Registry   string `json:"registry" gorm:"column:registry; default:''; NOT NULL"`
	Image      string `json:"image" gorm:"column:image; default:''; NOT NULL"`
	WorkflowID int    `json:"workflow_id,omitempty" gorm:"column:workflow_id; default:0; NOT NULL"`
	CIItems    []CI   `json:"ci_items,omitempty" gorm:"-"`
	CDItems    []CD   `json:"cd_items,omitempty" gorm:"-"`
}

type Workflow struct {
	ID       int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name     string `json:"name,omitempty" gorm:"column:name; default:''; NOT NULL"`
	Language string `json:"language,omitempty" gorm:"column:language; default:''; NOT NULL"`
	Workflow string `json:"workflow,omitempty" gorm:"column:workflow; default:''; NOT NULL"`
}

type CI struct {
	ID          int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	ServiceID   int    `json:"service_id,omitempty" gorm:"column:service_id; default:0; NOT NULL"`
}

type CD struct {
	ID          int    `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Description string `json:"description,omitempty" gorm:"column:description; default:''; NOT NULL"`
	ServiceID   int    `json:"service_id,omitempty" gorm:"column:service_id; default:0; NOT NULL"`
}
