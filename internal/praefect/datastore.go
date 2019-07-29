// Package praefect provides data models and datastore persistence abstractions
// for tracking the state of repository replicas.
//
// See original design discussion:
// https://gitlab.com/gitlab-org/gitaly/issues/1495
package praefect

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
)

var (
	// ErrPrimaryNotSet indicates the primary has not been set in the datastore
	ErrPrimaryNotSet = errors.New("primary is not set")
)

// JobState is an enum that indicates the state of a job
type JobState uint8

const (
	// JobStatePending is the initial job state when it is not yet ready to run
	// and may indicate recovery from a failure prior to the ready-state
	JobStatePending JobState = 1 << iota
	// JobStateReady indicates the job is now ready to proceed
	JobStateReady
	// JobStateInProgress indicates the job is being processed by a worker
	JobStateInProgress
	// JobStateComplete indicates the job is now complete
	JobStateComplete
	// JobStateCancelled indicates the job was cancelled. This can occur if the
	// job is no longer relevant (e.g. a node is moved out of a shard)
	JobStateCancelled
)

// ReplJob is an instance of a queued replication job. A replication job is
// meant for updating the repository so that it is synced with the primary
// copy. Scheduled indicates when a replication job should be performed.
type ReplJob struct {
	ID            uint64 // autoincrement ID
	TargetNodeID  int    // which node to replicate to?
	SourceStorage string
	Source        models.Repository // source for replication
	State         JobState
}

// replJobs provides sort manipulation behavior
type replJobs []ReplJob

func (rjs replJobs) Len() int      { return len(rjs) }
func (rjs replJobs) Swap(i, j int) { rjs[i], rjs[j] = rjs[j], rjs[i] }

// byJobID provides a comparator for sorting jobs
type byJobID struct{ replJobs }

func (b byJobID) Less(i, j int) bool { return b.replJobs[i].ID < b.replJobs[j].ID }

// Datastore is a data persistence abstraction for all of Praefect's
// persistence needs
type Datastore interface {
	ReplJobsDatastore
	ReplicasDatastore
}

// ReplicasDatastore manages accessing and setting which secondary replicas
// backup a repository
type ReplicasDatastore interface {
	GetReplicas(relativePath string) ([]models.StorageNode, error)

	GetStorageNode(nodeID int) (models.StorageNode, error)

	GetStorageNodes() ([]models.StorageNode, error)

	GetPrimary(relativePath string) (*models.StorageNode, error)

	SetPrimary(relativePath string, storageNodeID int) error

	AddReplica(relativePath string, storageNodeID int) error

	RemoveReplica(relativePath string, storageNodeID int) error

	GetRepository(relativePath string) (*models.Repository, error)
}

// ReplJobsDatastore represents the behavior needed for fetching and updating
// replication jobs from the datastore
type ReplJobsDatastore interface {
	// GetJobs fetches a list of chronologically ordered replication
	// jobs for the given storage replica. The returned list will be at most
	// count-length.
	GetJobs(flag JobState, nodeID int, count int) ([]ReplJob, error)

	// CreateReplicaJobs will create replication jobs for each secondary
	// replica of a repository known to the datastore. A set of replication job
	// ID's for the created jobs will be returned upon success.
	CreateReplicaReplJobs(relativePath string) ([]uint64, error)

	// UpdateReplJob updates the state of an existing replication job
	UpdateReplJob(jobID uint64, newState JobState) error
}

type jobRecord struct {
	sourceStorage string
	relativePath  string // project's relative path
	targetNodeID  int
	state         JobState
}

// MemoryDatastore is a simple datastore that isn't persisted to disk. It is
// only intended for early beta requirements and as a reference implementation
// for the eventual SQL implementation
type MemoryDatastore struct {
	jobs *struct {
		sync.RWMutex
		next    uint64
		records map[uint64]jobRecord // all jobs indexed by ID
	}

	storageNodes *struct {
		sync.RWMutex
		m map[int]models.StorageNode
	}

	repositories *struct {
		sync.RWMutex
		m map[string]models.Repository
	}
}

// NewMemoryDatastore returns an initialized in-memory datastore
func NewMemoryDatastore() *MemoryDatastore {
	return &MemoryDatastore{
		storageNodes: &struct {
			sync.RWMutex
			m map[int]models.StorageNode
		}{
			m: map[int]models.StorageNode{},
		},
		jobs: &struct {
			sync.RWMutex
			next    uint64
			records map[uint64]jobRecord // all jobs indexed by ID
		}{
			next:    0,
			records: map[uint64]jobRecord{},
		},
		repositories: &struct {
			sync.RWMutex
			m map[string]models.Repository
		}{
			m: map[string]models.Repository{},
		},
	}
}

// GetReplicas gets the secondaries for a shard based on the relative path
func (md *MemoryDatastore) GetReplicas(relativePath string) ([]models.StorageNode, error) {
	md.repositories.RLock()
	md.storageNodes.RLock()
	defer md.storageNodes.RUnlock()
	defer md.repositories.RUnlock()

	shard, ok := md.repositories.m[relativePath]
	if !ok {
		return nil, errors.New("shard not found")
	}

	return shard.Replicas, nil
}

// GetStorageNode gets all storage nodes
func (md *MemoryDatastore) GetStorageNode(nodeID int) (models.StorageNode, error) {
	md.storageNodes.RLock()
	defer md.storageNodes.RUnlock()

	node, ok := md.storageNodes.m[nodeID]
	if !ok {
		return models.StorageNode{}, errors.New("node not found")
	}

	return node, nil
}

// GetStorageNodes gets all storage nodes
func (md *MemoryDatastore) GetStorageNodes() ([]models.StorageNode, error) {
	md.storageNodes.RLock()
	defer md.storageNodes.RUnlock()

	var storageNodes []models.StorageNode
	for _, storageNode := range md.storageNodes.m {
		storageNodes = append(storageNodes, storageNode)
	}

	return storageNodes, nil
}

// GetPrimary gets the primary storage node for a shard of a repository relative path
func (md *MemoryDatastore) GetPrimary(relativePath string) (*models.StorageNode, error) {
	md.repositories.RLock()
	defer md.repositories.RUnlock()

	shard, ok := md.repositories.m[relativePath]
	if !ok {
		return nil, errors.New("shard not found")
	}

	storageNode, ok := md.storageNodes.m[shard.Primary.ID]
	if !ok {
		return nil, errors.New("node storage not found")
	}
	return &storageNode, nil

}

// SetPrimary sets the primary storagee node for a shard of a repository relative path
func (md *MemoryDatastore) SetPrimary(relativePath string, storageNodeID int) error {
	md.repositories.Lock()
	defer md.repositories.Unlock()

	shard, ok := md.repositories.m[relativePath]
	if !ok {
		return errors.New("shard not found")
	}

	storageNode, ok := md.storageNodes.m[storageNodeID]
	if !ok {
		return errors.New("node storage not found")
	}

	shard.Primary = storageNode

	md.repositories.m[relativePath] = shard
	return nil
}

// AddReplica adds a secondary to a shard of a repository relative path
func (md *MemoryDatastore) AddReplica(relativePath string, storageNodeID int) error {
	md.repositories.Lock()
	defer md.repositories.Unlock()

	shard, ok := md.repositories.m[relativePath]
	if !ok {
		return errors.New("shard not found")
	}

	storageNode, ok := md.storageNodes.m[storageNodeID]
	if !ok {
		return errors.New("node storage not found")
	}

	shard.Replicas = append(shard.Replicas, storageNode)

	md.repositories.m[relativePath] = shard
	return nil
}

// RemoveReplica removes a secondary from a shard of a repository relative path
func (md *MemoryDatastore) RemoveReplica(relativePath string, storageNodeID int) error {
	md.repositories.Lock()
	defer md.repositories.Unlock()

	shard, ok := md.repositories.m[relativePath]
	if !ok {
		return errors.New("shard not found")
	}

	var secondaries []models.StorageNode
	for _, secondary := range shard.Replicas {
		if secondary.ID != storageNodeID {
			secondaries = append(secondaries, secondary)
		}
	}

	shard.Replicas = secondaries
	md.repositories.m[relativePath] = shard
	return nil
}

// GetRepository gets the shard for a repository relative path
func (md *MemoryDatastore) GetRepository(relativePath string) (*models.Repository, error) {
	md.repositories.Lock()
	defer md.repositories.Unlock()

	shard, ok := md.repositories.m[relativePath]
	if !ok {
		return nil, errors.New("shard not found")
	}

	return &shard, nil
}

// ErrReplicasMissing indicates the repository does not have any backup
// replicas
var ErrReplicasMissing = errors.New("repository missing secondary replicas")

// GetJobs is a more general method to retrieve jobs of a certain state from the datastore
func (md *MemoryDatastore) GetJobs(state JobState, targetNodeID int, count int) ([]ReplJob, error) {
	md.jobs.RLock()
	defer md.jobs.RUnlock()

	var results []ReplJob

	for i, record := range md.jobs.records {
		// state is a bitmap that is a combination of one or more JobStates
		if record.state&state != 0 && record.targetNodeID == targetNodeID {
			job, err := md.replJobFromRecord(i, record)
			if err != nil {
				return nil, err
			}

			results = append(results, job)
			if len(results) >= count {
				break
			}
		}
	}

	sort.Sort(byJobID{results})

	return results, nil
}

// replJobFromRecord constructs a replication job from a record and by cross
// referencing the current shard for the project being replicated
func (md *MemoryDatastore) replJobFromRecord(jobID uint64, record jobRecord) (ReplJob, error) {
	return ReplJob{
		ID: jobID,
		Source: models.Repository{
			RelativePath: record.relativePath,
		},
		SourceStorage: record.sourceStorage,
		State:         record.state,
		TargetNodeID:  record.targetNodeID,
	}, nil
}

// ErrInvalidReplTarget indicates a targetStorage repository cannot be chosen because
// it fails preconditions for being replicatable
var ErrInvalidReplTarget = errors.New("targetStorage repository fails preconditions for replication")

// CreateReplicaReplJobs creates a replication job for each secondary that
// backs the specified repository. Upon success, the job IDs will be returned.
func (md *MemoryDatastore) CreateReplicaReplJobs(relativePath string) ([]uint64, error) {
	md.jobs.Lock()
	defer md.jobs.Unlock()

	if relativePath == "" {
		return nil, errors.New("invalid source repository")
	}

	shard, err := md.GetRepository(relativePath)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to find shard for project at relative path %q",
			relativePath,
		)
	}

	var jobIDs []uint64

	for _, secondary := range shard.Replicas {
		nextID := uint64(len(md.jobs.records) + 1)

		md.jobs.next++
		md.jobs.records[md.jobs.next] = jobRecord{
			targetNodeID:  secondary.ID,
			state:         JobStatePending,
			relativePath:  relativePath,
			sourceStorage: shard.Primary.Storage,
		}

		jobIDs = append(jobIDs, nextID)
	}

	return jobIDs, nil
}

// UpdateReplJob updates an existing replication job's state
func (md *MemoryDatastore) UpdateReplJob(jobID uint64, newState JobState) error {
	md.jobs.Lock()
	defer md.jobs.Unlock()

	job, ok := md.jobs.records[jobID]
	if !ok {
		return fmt.Errorf("job ID %d does not exist", jobID)
	}

	if newState == JobStateComplete || newState == JobStateCancelled {
		// remove the job to avoid filling up memory with unneeded job records
		delete(md.jobs.records, jobID)
		return nil
	}

	job.state = newState
	md.jobs.records[jobID] = job
	return nil
}
