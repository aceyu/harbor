package core

import (
	"fmt"
	common_models "github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/replication"
	"github.com/vmware/harbor/src/replication/models"
	"github.com/vmware/harbor/src/replication/replicator"
	"github.com/vmware/harbor/src/ui/utils"
)

type PullController struct {
	//Indicate whether the controller has been initialized or not
	initialized bool

	//Handle the replication work
	replicator replicator.Replicator
}

var (
	PullRepController Controller
)

func (ctl *PullController) Init() error {
	if ctl.initialized {
		return nil
	}
	ctl.initialized = true

	return nil
}

func NewPullController(cfg ControllerConfig) *PullController {
	//Controller refer the default instances
	ctl := &PullController{}
	ctl.replicator = replicator.NewPullReplicator(utils.GetJobServiceClient())

	return ctl
}

func (ctl *PullController) Replicate(policyID int64, metadata ...map[string]interface{}) error {

	if len(metadata) == 0 {
		return fmt.Errorf("metadata not found")
	}

	meta := metadata[0]

	srcs := []*common_models.RepTarget{}

	var url string
	var insecure bool
	if u, ok := meta["url"].(string); ok {
		url = u
	}
	if i, ok := meta["insecure"].(bool); ok {
		insecure = i
	}
	if len(url) == 0 {
		return fmt.Errorf("target not found")
	}

	src := &common_models.RepTarget{
		URL:      url,
		Insecure: insecure,
	}
	srcs = append(srcs, src)

	var repository string
	if r, ok := meta["repository"].(string); ok {
		repository = r
	} else {
		return fmt.Errorf("repository not found")
	}
	candidates := []models.FilterItem{}
	canMeta := make(map[string]interface{})
	if u, ok := meta["pull_username"].(string); ok {
		canMeta["pull_username"] = u
	}
	candidate := &models.FilterItem{
		Kind:      replication.FilterItemKindRepository,
		Value:     repository,
		Operation: common_models.RepOpTransfer,
		Metadata:  canMeta,
	}
	candidates = append(candidates, *candidate)
	// submit the replication
	return ctl.replicator.Replicate(&replicator.Replication{
		PolicyID:   policyID,
		Candidates: candidates,
		Targets:    srcs,
	})
}

func (ctl *PullController) GetPolicies(models.QueryParameter) (*models.ReplicationPolicyQueryResult, error) {
	return nil, nil
}
func (ctl *PullController) GetPolicy(int64) (models.ReplicationPolicy, error) {
	return models.ReplicationPolicy{}, nil
}
func (ctl *PullController) CreatePolicy(models.ReplicationPolicy) (int64, error) {
	return -1, nil
}
func (ctl *PullController) UpdatePolicy(models.ReplicationPolicy) error {
	return nil
}
func (ctl *PullController) RemovePolicy(int64) error {
	return nil
}
