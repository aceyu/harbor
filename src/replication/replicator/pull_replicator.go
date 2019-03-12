package replicator

import (
	"fmt"
	"strings"

	"github.com/vmware/harbor/src/common/dao"
	common_job "github.com/vmware/harbor/src/common/job"
	job_models "github.com/vmware/harbor/src/common/job/models"
	common_models "github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/config"
)

type PullReplicator struct {
	client common_job.Client
}

func NewPullReplicator(client common_job.Client) *PullReplicator {
	return &PullReplicator{
		client: client,
	}
}

func (d *PullReplicator) Replicate(replication *Replication) error {
	repositories := map[string][]string{}
	// TODO the operation of all candidates are same for now. Update it after supporting
	// replicate deletion
	operation := ""
	var pull_username interface{}
	for _, candidate := range replication.Candidates {
		strs := strings.SplitN(candidate.Value, ":", 2)
		repositories[strs[0]] = append(repositories[strs[0]], strs[1])
		operation = candidate.Operation

		meta := candidate.Metadata
		pull_username = meta["pull_username"]
	}
	username := ""
	if u, ok := pull_username.(string); ok {
		username = u
	}
	for _, src := range replication.Targets {
		for repository, tags := range repositories {
			// create job in database
			id, err := dao.AddRepJob(common_models.RepJob{
				PolicyID:   replication.PolicyID,
				Repository: repository,
				TagList:    tags,
				Operation:  username,
			})
			if err != nil {
				return err
			}

			// submit job to jobservice
			log.Debugf("submiting pull replication job to jobservice, repository: %s, tags: %v, operation: %s, target: %s",
				repository, tags, operation, src.URL)
			job := &job_models.JobData{
				Metadata: &job_models.JobMetadata{
					JobKind: common_job.JobKindGeneric,
				},
				StatusHook: fmt.Sprintf("%s/service/notifications/jobs/replication/%d",
					config.InternalUIURL(), id),
			}

			if operation == common_models.RepOpTransfer {
				url, err := config.ExtEndpoint()
				if err != nil {
					return err
				}
				job.Name = common_job.ImageTransfer
				job.Parameters = map[string]interface{}{
					"repository":            repository,
					"tags":                  tags,
					"src_registry_url":      src.URL,
					"src_registry_insecure": src.Insecure,
					"dst_registry_url":      url,
					"pull_username":         pull_username,
					"dst_registry_insecure": true,
				}
			}

			uuid, err := d.client.SubmitJob(job)
			if err != nil {
				if er := dao.UpdateRepJobStatus(id, common_models.JobError); er != nil {
					log.Errorf("failed to update the status of job %d: %s", id, er)
				}
				return err
			}

			// create the mapping relationship between the jobs in database and jobservice
			if err = dao.SetRepJobUUID(id, uuid); err != nil {
				return err
			}
		}
	}
	return nil
}
