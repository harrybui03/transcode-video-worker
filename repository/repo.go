package repository

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"worker-transcode/constant"
	"worker-transcode/entities"
)

type JobRepository interface {
	Transaction(ctx context.Context, callback func(ctx context.Context) error, opts ...*sql.TxOptions) error
	GetDB() *gorm.DB
	FindJobById(ctx context.Context, id uuid.UUID) (*entities.Job, error)
	UpdateStatusJob(context context.Context, status constant.JobStatus, id uuid.UUID) error
	UpdateLessonVideoURL(ctx context.Context, lessonId uuid.UUID, url string) error
}

type repo struct {
	db *gorm.DB
}

func (r *repo) UpdateLessonVideoURL(ctx context.Context, lessonId uuid.UUID, url string) error {
	lesson := &entities.Lesson{}
	err := r.GetDB().Model(lesson).Where("id = ?", lessonId).Update("video_url", url).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) FindJobById(ctx context.Context, id uuid.UUID) (*entities.Job, error) {
	job := &entities.Job{}
	err := r.GetDB().First(job, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (r *repo) UpdateStatusJob(context context.Context, status constant.JobStatus, id uuid.UUID) error {
	job := &entities.Job{}
	err := r.GetDB().First(job, "id = ?", id).Error
	if err != nil {
		return err
	}
	job.Status = status
	err = r.GetDB().Save(job).Error
	if err != nil {
		return err
	}
	return nil
}

func NewRepo(db *sql.DB) JobRepository {
	gormDB, _ := gorm.Open(postgres.New(postgres.Config{
		Conn: db}),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		},
	)
	return &repo{
		db: gormDB,
	}
}

func (r *repo) GetDB() *gorm.DB {
	return r.db
}

func (r *repo) Transaction(ctx context.Context, callback func(ctx context.Context) error, opts ...*sql.TxOptions) error {
	return r.GetDB().Transaction(func(tx *gorm.DB) error {
		err := callback(ctx)
		if err != nil {
			return err
		}
		return nil
	}, opts...)
}
