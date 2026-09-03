package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type listBackend interface {
	Push(context.Context, string, string) error
	Move(context.Context, string, string, time.Duration) (string, error)
	Remove(context.Context, string, string) error
}

type RedisQueue struct {
	backend                      listBackend
	main, processing, deadLetter string
	maxAttempts                  int
}

func NewRedis(client *redis.Client, main, processing, deadLetter string, maxAttempts int) *RedisQueue {
	return newRedisQueue(redisListBackend{client: client}, main, processing, deadLetter, maxAttempts)
}

func newRedisQueue(backend listBackend, main, processing, deadLetter string, maxAttempts int) *RedisQueue {
	return &RedisQueue{backend: backend, main: main, processing: processing, deadLetter: deadLetter, maxAttempts: maxAttempts}
}

func (q *RedisQueue) Enqueue(ctx context.Context, job domain.DocumentJob) error {
	if err := validateJob(job); err != nil {
		return err
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if err := q.backend.Push(ctx, q.main, string(payload)); err != nil {
		return apperrors.Wrap(apperrors.ErrQueueUnavailable, err)
	}
	return nil
}

func (q *RedisQueue) Consume(ctx context.Context) (domain.ClaimedDocumentJob, error) {
	payload, err := q.backend.Move(ctx, q.main, q.processing, 0)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrQueueUnavailable, err)
	}
	var job domain.DocumentJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		_ = q.backend.Remove(ctx, q.processing, payload)
		return nil, fmt.Errorf("decode document job: %w", err)
	}
	if err := validateJob(job); err != nil {
		_ = q.backend.Remove(ctx, q.processing, payload)
		return nil, err
	}
	return &claimedJob{queue: q, job: job, payload: payload}, nil
}

func validateJob(job domain.DocumentJob) error {
	if job.DocumentID <= 0 || job.Attempt <= 0 {
		return apperrors.ErrQueueInvalidJob
	}
	return nil
}

type claimedJob struct {
	queue   *RedisQueue
	job     domain.DocumentJob
	payload string
}

func (j *claimedJob) Job() domain.DocumentJob { return j.job }
func (j *claimedJob) Ack(ctx context.Context) error {
	return j.queue.backend.Remove(ctx, j.queue.processing, j.payload)
}
func (j *claimedJob) Retry(ctx context.Context, _ error) error {
	destination := j.queue.main
	next := j.job
	if j.job.Attempt >= j.queue.maxAttempts {
		destination = j.queue.deadLetter
	} else {
		next.Attempt++
	}
	payload, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := j.queue.backend.Remove(ctx, j.queue.processing, j.payload); err != nil {
		return err
	}
	if err := j.queue.backend.Push(ctx, destination, string(payload)); err != nil {
		return apperrors.Wrap(apperrors.ErrQueueUnavailable, err)
	}
	return nil
}

type redisListBackend struct{ client *redis.Client }

func (b redisListBackend) Push(ctx context.Context, key, payload string) error {
	return b.client.LPush(ctx, key, payload).Err()
}
func (b redisListBackend) Move(ctx context.Context, source, destination string, timeout time.Duration) (string, error) {
	return b.client.BLMove(ctx, source, destination, "RIGHT", "LEFT", timeout).Result()
}
func (b redisListBackend) Remove(ctx context.Context, key, payload string) error {
	return b.client.LRem(ctx, key, 1, payload).Err()
}
