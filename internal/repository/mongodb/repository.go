package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const defaultOperationTimeout = 5 * time.Second

type Repository struct {
	client  *mongo.Client
	db      *mongo.Database
	timeout time.Duration
}

func New(ctx context.Context, uri string, database string) (*Repository, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongodb uri cannot be empty")
	}
	if database == "" {
		return nil, fmt.Errorf("mongodb database cannot be empty")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	return &Repository{
		client:  client,
		db:      client.Database(database),
		timeout: defaultOperationTimeout,
	}, nil
}

func (r *Repository) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

func (r *Repository) AddNode(node *domain.Node) error {
	if node.Name == "" {
		return fmt.Errorf("node name cannot be empty")
	}
	return r.replaceOne("nodes", bson.M{"name": node.Name}, node)
}

func (r *Repository) GetNode(name string) (*domain.Node, error) {
	var node domain.Node
	if err := r.findOne("nodes", bson.M{"name": name}, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *Repository) ListNodes() ([]*domain.Node, error) {
	return findMany[domain.Node](r, "nodes", bson.M{})
}

func (r *Repository) AddGPU(gpu *domain.GPU) error {
	if gpu.ID == "" {
		return fmt.Errorf("gpu id cannot be empty")
	}
	return r.replaceOne("gpus", bson.M{"id": gpu.ID}, gpu)
}

func (r *Repository) GetGPU(id string) (*domain.GPU, error) {
	var gpu domain.GPU
	if err := r.findOne("gpus", bson.M{"id": id}, &gpu); err != nil {
		return nil, err
	}
	return &gpu, nil
}

func (r *Repository) ListGPUs(nodeName string) ([]*domain.GPU, error) {
	filter := bson.M{}
	if nodeName != "" {
		filter["nodename"] = nodeName
	}
	return findMany[domain.GPU](r, "gpus", filter)
}

func (r *Repository) AddJob(job *domain.Job) error {
	if job.ID == "" {
		return fmt.Errorf("job id cannot be empty")
	}
	return r.replaceOne("jobs", bson.M{"id": job.ID}, job)
}

func (r *Repository) GetJob(id string) (*domain.Job, error) {
	var job domain.Job
	if err := r.findOne("jobs", bson.M{"id": id}, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *Repository) ListJobs() ([]*domain.Job, error) {
	return findMany[domain.Job](r, "jobs", bson.M{})
}

func (r *Repository) DeleteJob(id string) error {
	ctx, cancel := r.operationContext()
	defer cancel()

	result, err := r.db.Collection("jobs").DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("job %s not found", id)
	}
	return nil
}

func (r *Repository) AddQueue(queue *domain.Queue) error {
	if queue.Name == "" {
		return fmt.Errorf("queue name cannot be empty")
	}
	return r.replaceOne("queues", bson.M{"name": queue.Name}, queue)
}

func (r *Repository) GetQueue(name string) (*domain.Queue, error) {
	var queue domain.Queue
	if err := r.findOne("queues", bson.M{"name": name}, &queue); err != nil {
		return nil, err
	}
	return &queue, nil
}

func (r *Repository) ListQueues() ([]*domain.Queue, error) {
	return findMany[domain.Queue](r, "queues", bson.M{})
}

func (r *Repository) DeleteQueue(name string) error {
	ctx, cancel := r.operationContext()
	defer cancel()

	result, err := r.db.Collection("queues").DeleteOne(ctx, bson.M{"name": name})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("queue %s not found", name)
	}
	return nil
}

func (r *Repository) AddReservation(reservation *domain.NodeReservation) error {
	if reservation.ID == "" {
		return fmt.Errorf("reservation id cannot be empty")
	}
	return r.replaceOne("reservations", bson.M{"id": reservation.ID}, reservation)
}

func (r *Repository) GetReservation(id string) (*domain.NodeReservation, error) {
	var reservation domain.NodeReservation
	if err := r.findOne("reservations", bson.M{"id": id}, &reservation); err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (r *Repository) ListReservations() ([]*domain.NodeReservation, error) {
	return findMany[domain.NodeReservation](r, "reservations", bson.M{})
}

func (r *Repository) AddUser(user *domain.User) error {
	if user.Alias == "" {
		return fmt.Errorf("user alias cannot be empty")
	}
	return r.replaceOne("users", bson.M{"alias": user.Alias}, user)
}

func (r *Repository) GetUser(alias string) (*domain.User, error) {
	var user domain.User
	if err := r.findOne("users", bson.M{"alias": alias}, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) ListUsers() ([]*domain.User, error) {
	return findMany[domain.User](r, "users", bson.M{})
}

func (r *Repository) AddProject(project *domain.Project) error {
	if project.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	return r.replaceOne("projects", bson.M{"name": project.Name}, project)
}

func (r *Repository) GetProject(name string) (*domain.Project, error) {
	var project domain.Project
	if err := r.findOne("projects", bson.M{"name": name}, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *Repository) ListProjects() ([]*domain.Project, error) {
	return findMany[domain.Project](r, "projects", bson.M{})
}

func (r *Repository) UpdateGPUAllocation(gpuID, jobID string) error {
	if jobID != "" {
		return r.AllocateGPUs([]string{gpuID}, jobID)
	}
	return r.ReleaseGPUs([]string{gpuID})
}

func (r *Repository) AllocateGPUs(gpuIDs []string, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("job id cannot be empty")
	}
	if len(gpuIDs) == 0 {
		return nil
	}

	ctx, cancel := r.operationContext()
	defer cancel()

	filter := bson.M{"id": bson.M{"$in": gpuIDs}, "status": domain.GPUStatusAvailable}
	update := bson.M{"$set": bson.M{"status": domain.GPUStatusAllocated, "allocatedto": jobID}}
	result, err := r.db.Collection("gpus").UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.ModifiedCount != int64(len(gpuIDs)) {
		_, _ = r.db.Collection("gpus").UpdateMany(ctx, bson.M{"allocatedto": jobID}, bson.M{"$set": bson.M{"status": domain.GPUStatusAvailable, "allocatedto": ""}})
		return fmt.Errorf("not all requested GPUs are available")
	}
	return nil
}

func (r *Repository) ReleaseGPUs(gpuIDs []string) error {
	if len(gpuIDs) == 0 {
		return nil
	}

	ctx, cancel := r.operationContext()
	defer cancel()

	result, err := r.db.Collection("gpus").UpdateMany(ctx, bson.M{"id": bson.M{"$in": gpuIDs}}, bson.M{"$set": bson.M{"status": domain.GPUStatusAvailable, "allocatedto": ""}})
	if err != nil {
		return err
	}
	if result.MatchedCount != int64(len(gpuIDs)) {
		return fmt.Errorf("one or more GPUs were not found")
	}
	return nil
}

func (r *Repository) replaceOne(collection string, filter bson.M, document interface{}) error {
	ctx, cancel := r.operationContext()
	defer cancel()

	_, err := r.db.Collection(collection).ReplaceOne(ctx, filter, document, options.Replace().SetUpsert(true))
	return err
}

func (r *Repository) findOne(collection string, filter bson.M, target interface{}) error {
	ctx, cancel := r.operationContext()
	defer cancel()

	err := r.db.Collection(collection).FindOne(ctx, filter).Decode(target)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("document not found")
	}
	return err
}

func findMany[T any](r *Repository, collection string, filter bson.M) ([]*T, error) {
	ctx, cancel := r.operationContext()
	defer cancel()

	cursor, err := r.db.Collection(collection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*T
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) operationContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.timeout)
}
