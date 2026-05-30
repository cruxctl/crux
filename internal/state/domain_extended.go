package store

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

// --- Session Store ---

func (d *domainStore) CreateSession(ctx context.Context, sess cruxapi.Session) error {
	if sess.ID == "" {
		return fmt.Errorf("session id is required")
	}
	if sess.StartedAt.IsZero() {
		sess.StartedAt = time.Now().UTC()
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("sessions", sess.ID+".json"), data)
}

func (d *domainStore) GetSession(ctx context.Context, id string) (cruxapi.Session, error) {
	data, err := d.store.Get(ctx, filepath.Join("sessions", id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.Session{}, ErrNotFound
		}
		return cruxapi.Session{}, err
	}
	var sess cruxapi.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return cruxapi.Session{}, err
	}
	return sess, nil
}

func (d *domainStore) ListSessions(ctx context.Context) ([]cruxapi.Session, error) {
	objs, err := d.store.List(ctx, "sessions")
	if err != nil {
		return nil, err
	}
	var sessions []cruxapi.Session
	for _, obj := range objs {
		if !strings.HasSuffix(obj.Path, ".json") {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var sess cruxapi.Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.After(sessions[j].StartedAt)
	})
	return sessions, nil
}

func (d *domainStore) UpdateSession(ctx context.Context, sess cruxapi.Session) error {
	_, err := d.GetSession(ctx, sess.ID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("sessions", sess.ID+".json"), data)
}

// --- AOS Event Store ---

func (d *domainStore) AppendAOSEvent(ctx context.Context, event cruxapi.AOSEvent) error {
	if event.EventID == "" {
		event.EventID = cruxapi.NewID("aos")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return d.store.Append(ctx, "aos/events.jsonl", data)
}

func (d *domainStore) ListAOSEvents(ctx context.Context) ([]cruxapi.AOSEvent, error) {
	data, err := d.store.Get(ctx, "aos/events.jsonl")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []cruxapi.AOSEvent
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var e cruxapi.AOSEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	return events, nil
}

func (d *domainStore) GetAOSEvent(ctx context.Context, id string) (cruxapi.AOSEvent, error) {
	data, err := d.store.Get(ctx, "aos/events.jsonl")
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.AOSEvent{}, ErrNotFound
		}
		return cruxapi.AOSEvent{}, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var e cruxapi.AOSEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.EventID == id {
			return e, nil
		}
	}
	return cruxapi.AOSEvent{}, ErrNotFound
}

// --- Policy Store ---

func (d *domainStore) UpsertPolicy(ctx context.Context, policy cruxapi.PolicyProfile) error {
	if policy.Metadata.ID == "" {
		return fmt.Errorf("policy id is required")
	}
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("policies", policy.Metadata.ID+".json"), data)
}

func (d *domainStore) GetPolicy(ctx context.Context, id string) (cruxapi.PolicyProfile, error) {
	data, err := d.store.Get(ctx, filepath.Join("policies", id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.PolicyProfile{}, ErrNotFound
		}
		return cruxapi.PolicyProfile{}, err
	}
	var policy cruxapi.PolicyProfile
	if err := json.Unmarshal(data, &policy); err != nil {
		return cruxapi.PolicyProfile{}, err
	}
	return policy, nil
}

func (d *domainStore) ListPolicies(ctx context.Context) ([]cruxapi.PolicyProfile, error) {
	objs, err := d.store.List(ctx, "policies")
	if err != nil {
		return nil, err
	}
	var policies []cruxapi.PolicyProfile
	for _, obj := range objs {
		if !strings.HasSuffix(obj.Path, ".json") {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var policy cruxapi.PolicyProfile
		if err := json.Unmarshal(data, &policy); err != nil {
			continue
		}
		policies = append(policies, policy)
	}
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Metadata.ID < policies[j].Metadata.ID
	})
	return policies, nil
}

func (d *domainStore) DeletePolicy(ctx context.Context, id string) error {
	_, err := d.GetPolicy(ctx, id)
	if err != nil {
		return err
	}
	return d.store.Delete(ctx, filepath.Join("policies", id+".json"))
}

// --- Approval Store ---

func (d *domainStore) CreateApproval(ctx context.Context, rec cruxapi.ApprovalRecord) error {
	if rec.ID == "" {
		rec.ID = cruxapi.NewID("apr")
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("approvals", rec.ID+".json"), data)
}

func (d *domainStore) GetApproval(ctx context.Context, id string) (cruxapi.ApprovalRecord, error) {
	data, err := d.store.Get(ctx, filepath.Join("approvals", id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.ApprovalRecord{}, ErrNotFound
		}
		return cruxapi.ApprovalRecord{}, err
	}
	var rec cruxapi.ApprovalRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return cruxapi.ApprovalRecord{}, err
	}
	return rec, nil
}

func (d *domainStore) ListApprovals(ctx context.Context) ([]cruxapi.ApprovalRecord, error) {
	objs, err := d.store.List(ctx, "approvals")
	if err != nil {
		return nil, err
	}
	var recs []cruxapi.ApprovalRecord
	for _, obj := range objs {
		if !strings.HasSuffix(obj.Path, ".json") {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var rec cruxapi.ApprovalRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		recs = append(recs, rec)
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].CreatedAt.After(recs[j].CreatedAt)
	})
	return recs, nil
}

func (d *domainStore) UpdateApproval(ctx context.Context, rec cruxapi.ApprovalRecord) error {
	_, err := d.GetApproval(ctx, rec.ID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("approvals", rec.ID+".json"), data)
}

// --- Job Store ---

func (d *domainStore) CreateJob(ctx context.Context, job cruxapi.Job) error {
	if job.ID == "" {
		return fmt.Errorf("job id is required")
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("jobs", job.ID+".json"), data)
}

func (d *domainStore) GetJob(ctx context.Context, id string) (cruxapi.Job, error) {
	data, err := d.store.Get(ctx, filepath.Join("jobs", id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.Job{}, ErrNotFound
		}
		return cruxapi.Job{}, err
	}
	var job cruxapi.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return cruxapi.Job{}, err
	}
	return job, nil
}

func (d *domainStore) ListJobs(ctx context.Context) ([]cruxapi.Job, error) {
	objs, err := d.store.List(ctx, "jobs")
	if err != nil {
		return nil, err
	}
	var jobs []cruxapi.Job
	for _, obj := range objs {
		if !strings.HasSuffix(obj.Path, ".json") {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var job cruxapi.Job
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ID < jobs[j].ID
	})
	return jobs, nil
}

func (d *domainStore) DeleteJob(ctx context.Context, id string) error {
	_, err := d.GetJob(ctx, id)
	if err != nil {
		return err
	}
	return d.store.Delete(ctx, filepath.Join("jobs", id+".json"))
}

// --- Machine Store ---

func (d *domainStore) CreateMachine(ctx context.Context, m cruxapi.Machine) error {
	if m.ID == "" {
		return fmt.Errorf("machine id is required")
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, filepath.Join("machines", m.ID+".json"), data)
}

func (d *domainStore) GetMachine(ctx context.Context, id string) (cruxapi.Machine, error) {
	data, err := d.store.Get(ctx, filepath.Join("machines", id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.Machine{}, ErrNotFound
		}
		return cruxapi.Machine{}, err
	}
	var m cruxapi.Machine
	if err := json.Unmarshal(data, &m); err != nil {
		return cruxapi.Machine{}, err
	}
	return m, nil
}

func (d *domainStore) ListMachines(ctx context.Context) ([]cruxapi.Machine, error) {
	objs, err := d.store.List(ctx, "machines")
	if err != nil {
		return nil, err
	}
	var machines []cruxapi.Machine
	for _, obj := range objs {
		if !strings.HasSuffix(obj.Path, ".json") {
			continue
		}
		data, err := d.store.Get(ctx, obj.Path)
		if err != nil {
			continue
		}
		var m cruxapi.Machine
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		machines = append(machines, m)
	}
	sort.Slice(machines, func(i, j int) bool {
		return machines[i].ID < machines[j].ID
	})
	return machines, nil
}

// --- Usage Limits Store ---

func (d *domainStore) GetUsageLimits(ctx context.Context) (cruxapi.UsageLimits, error) {
	data, err := d.store.Get(ctx, "config/usage-limits.json")
	if err != nil {
		if os.IsNotExist(err) {
			return cruxapi.UsageLimits{}, nil
		}
		return cruxapi.UsageLimits{}, err
	}
	var limits cruxapi.UsageLimits
	if err := json.Unmarshal(data, &limits); err != nil {
		return cruxapi.UsageLimits{}, err
	}
	return limits, nil
}

func (d *domainStore) SetUsageLimits(ctx context.Context, limits cruxapi.UsageLimits) error {
	data, err := json.Marshal(limits)
	if err != nil {
		return err
	}
	return d.store.Put(ctx, "config/usage-limits.json", data)
}
