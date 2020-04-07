package teamcity

//TriggerScheduleOptions represent options for configuring a scheduled build trigger
type TriggerScheduleOptions struct {
	QueueOptimization                   bool                       `prop:"enableQueueOptimization" force:""`
	PromoteWatchedBuild                 bool                       `prop:"promoteWatchedBuild" force:""`
	BuildWithPendingChangesOnly         bool                       `prop:"triggerBuildWithPendingChangesOnly" force:""`
	TriggerIfWatchedBuildChanges        bool                       `prop:"triggerBuildIfWatchedBuildChanges"`
	BuildOnAllCompatibleAgents          bool                       `prop:"triggerBuildOnAllCompatibleAgents"`
	RevisionRuleSourceBuildID           string                     `prop:"revisionRuleDependsOn"`
	RevisionRule                        ArtifactDependencyRevision `prop:"revisionRule"`
	RevisionRuleBuildBranch             string                     `prop:"revisionRuleBuildBranch"`
	EnforceCleanCheckout                bool                       `prop:"enforceCleanCheckout"`
	EnforceCleanCheckoutForDependencies bool                       `prop:"enforceCleanCheckoutForDependencies"`
}

//NewTriggerScheduleOptions returns a TriggerScheduleOptions with default values
func NewTriggerScheduleOptions() *TriggerScheduleOptions {
	return &TriggerScheduleOptions{
		QueueOptimization:           true,
		PromoteWatchedBuild:         true,
		BuildWithPendingChangesOnly: true,
		RevisionRuleBuildBranch:     "<default>",
		RevisionRule:                LatestFinishedBuild,
	}
}

func (o *TriggerScheduleOptions) properties() *Properties {
	return serializeToProperties(o)
}

func (p *Properties) triggerScheduleOptions() *TriggerScheduleOptions {
	var out TriggerScheduleOptions

	fillStructFromProperties(&out, p)

	return &out
}
