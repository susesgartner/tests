package qase

const (
	QaseTokenEnvVar         = "QASE_AUTOMATION_TOKEN"
	TestRunEnvVar           = "QASE_TEST_RUN_ID"
	TestRunNameEnvVar       = "TEST_RUN_NAME"
	RancherManagerProjectID = "RM"
	AutomationSuiteID       = int32(554)
	AutomationTestNameID    = 15
	MultiSubTestPattern     = `(\w+/\w+/\w+){1,}`
	SubtestPattern          = `(\w+/\w+){1,1}`
	PassStatus              = "pass"
	SkipStatus              = "skip"
	FailStatus              = "fail"
	RecurringRunID          = 1
	RunSourceID             = 16
	TestResultsJSON         = "results.json"
	TestSource              = "GoValidation"
	TestSourceID            = 14
)
