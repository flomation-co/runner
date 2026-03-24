package runner

import "time"

type Flo struct {
	ID                 string     `json:"id" db:"id"`
	Name               string     `json:"name" db:"name"`
	OrganisationID     *string    `json:"organisation_id,omitempty" db:"organisation_id"`
	AuthorID           *string    `json:"author_id,omitempty" db:"author_id"`
	CreatedAt          *time.Time `json:"created_at" db:"created_at"`
	Scale              float32    `json:"scale" db:"scale"`
	XPosition          float32    `json:"x" db:"x"`
	YPosition          float32    `json:"y" db:"y"`
	ExecutionCount     int64      `json:"execution_count" db:"execution_count"`
	LastRun            *string    `json:"last_run" db:"last_run"`
	Duration           *int64     `json:"duration" db:"duration"`
	DurationAdditional *int64     `json:"duration_additional" db:"duration_additional"`
	LastExecution      *Execution `json:"last_execution" db:"last_execution"`
	EnvironmentID      *string    `json:"environment_id"`
}

type Execution struct {
	ID               string      `json:"id" db:"id"`
	FloID            string      `json:"flo_id" db:"flo_id"`
	Name             string      `json:"name" db:"name"`
	OwnerID          string      `json:"owner_id" db:"owner_id"`
	OrganisationID   *string     `json:"organisation_id" db:"organisation_id"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        *time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAd      *time.Time  `json:"completed_ad" db:"completed_at"`
	TriggeredBy      *string     `json:"triggered_by" db:"triggered_by"`
	ExecutionStatus  string      `json:"execution_status" db:"execution_status"`
	CompletionStatus string      `json:"completion_status" db:"completion_status"`
	Sequence         int64       `json:"sequence" db:"sequence"`
	Data             interface{} `json:"data" db:"data"`
	RunnerID         *string     `json:"runner_id" db:"runner_id"`
	TriggerType      *string     `json:"trigger_type,omitempty"`
	AuthorEmail      *string     `json:"author_email,omitempty"`
	TriggererEmail   *string     `json:"triggerer_email,omitempty"`
	EntryNodeID      *string     `json:"entry_node_id,omitempty"`
}

type PendingExecution struct {
	Flow      Flo         `json:"flo"`
	Execution Execution   `json:"execution"`
	Data      interface{} `json:"data"`
}

type ExecutionResult struct {
	HasErrored bool        `json:"has_errored"`
	State      interface{} `json:"state"`
}
