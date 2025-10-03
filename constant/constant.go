package constant

type JobStatus string

const (
	JobStatusPending   JobStatus = "PENDING"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusFailed    JobStatus = "FAILED"
	JobStatusCompleted JobStatus = "COMPLETED"
)

type JobType string

const (
	JobTypeTranscoder JobType = "transcoder"
)

type Environment string

const (
	EnvironmentProduction Environment = "production"
	EnvironmentStaging    Environment = "staging"
	EnvironmentDevelop    Environment = "develop"
)

func (e Environment) String() string {
	return string(e)
}
