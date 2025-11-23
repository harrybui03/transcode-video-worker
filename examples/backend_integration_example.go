package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

// Example: How Backend should integrate with the Recording Merge Worker

// RecordingMergeMessage represents the message to send to recording_merge_queue
type RecordingMergeMessage struct {
	JobId        uuid.UUID `json:"jobId"`
	LessonId     uuid.UUID `json:"lessonId"`
	ChunkFolder  string    `json:"chunkFolder"`
	TotalChunks  int       `json:"totalChunks"`
	OutputPrefix string    `json:"outputPrefix"`
}

// Job represents a job record in the database
type Job struct {
	ID         uuid.UUID
	EntityId   uuid.UUID // LessonId
	EntityType string    // "lesson"
	Status     string    // "PENDING"
	JobType    string    // "recording_merge"
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Example1: Complete workflow when all chunks are uploaded
func ExampleCompleteRecordingWorkflow(ctx context.Context, conn *amqp.Connection, lessonId uuid.UUID, totalChunks int) error {
	// Step 1: Create job record in database
	jobId := uuid.New()
	job := Job{
		ID:         jobId,
		EntityId:   lessonId,
		EntityType: "lesson",
		Status:     "PENDING",
		JobType:    "recording_merge",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Save job to database
	// db.Create(&job)
	fmt.Printf("Created job: %s\n", jobId)

	// Step 2: Prepare message for worker
	message := RecordingMergeMessage{
		JobId:        jobId,
		LessonId:     lessonId,
		ChunkFolder:  fmt.Sprintf("recordings/%s/", lessonId),
		TotalChunks:  totalChunks,
		OutputPrefix: fmt.Sprintf("lessons/%s", lessonId),
	}

	// Step 3: Publish message to RabbitMQ
	err := publishRecordingMergeMessage(ctx, conn, message)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Printf("Published recording merge message for lesson %s\n", lessonId)
	return nil
}

// publishRecordingMergeMessage sends message to recording_merge_queue
func publishRecordingMergeMessage(ctx context.Context, conn *amqp.Connection, message RecordingMergeMessage) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declare exchange (should match worker configuration)
	exchangeName := "recording_exchange"
	err = ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return err
	}

	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish message
	routingKey := "recording.merge.request"
	err = ch.PublishWithContext(
		ctx,
		exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Make message persistent
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// Example2: Chunk upload workflow
func ExampleChunkUploadWorkflow(lessonId uuid.UUID, chunkNumber int, chunkData []byte) error {
	// This runs each time frontend uploads a chunk

	// Step 1: Save chunk to MinIO
	objectPath := fmt.Sprintf("recordings/%s/chunk-%03d.webm", lessonId, chunkNumber)
	// minioClient.PutObject(ctx, bucket, objectPath, bytes.NewReader(chunkData), ...)

	// Step 2: Save chunk metadata to database (optional, for tracking)
	// recording := Recording{
	//     ID:          uuid.New(),
	//     LessonId:    lessonId,
	//     ChunkNumber: chunkNumber,
	//     ChunkPath:   objectPath,
	//     Status:      "uploaded",
	//     CreatedAt:   time.Now(),
	//     UpdatedAt:   time.Now(),
	// }
	// db.Create(&recording)

	fmt.Printf("Uploaded chunk %d for lesson %s to %s\n", chunkNumber, lessonId, objectPath)
	return nil
}

// Example3: Check job status
func ExampleCheckJobStatus(jobId uuid.UUID) (string, error) {
	// Query database for job status
	// var job Job
	// db.First(&job, "id = ?", jobId)
	// return job.Status, nil

	// Status values:
	// - PENDING: Job is waiting to be processed
	// - PROCESSING: Worker is currently processing the job
	// - COMPLETED: Job finished successfully
	// - FAILED: Job failed and won't be retried

	return "COMPLETED", nil
}

// Example4: Complete API endpoint structure
type CompleteRecordingRequest struct {
	LessonId    uuid.UUID `json:"lessonId"`
	TotalChunks int       `json:"totalChunks"`
}

type CompleteRecordingResponse struct {
	JobId   uuid.UUID `json:"jobId"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

func ExampleAPIEndpoint(ctx context.Context, req CompleteRecordingRequest) (*CompleteRecordingResponse, error) {
	// This endpoint is called when frontend completes recording

	// Validate that all chunks exist in MinIO
	// ... check if all chunks are uploaded ...

	// Create job and trigger merge
	jobId := uuid.New()
	job := Job{
		ID:         jobId,
		EntityId:   req.LessonId,
		EntityType: "lesson",
		Status:     "PENDING",
		JobType:    "recording_merge",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Save to database
	// db.Create(&job)

	// Publish message
	message := RecordingMergeMessage{
		JobId:        jobId,
		LessonId:     req.LessonId,
		ChunkFolder:  fmt.Sprintf("recordings/%s/", req.LessonId),
		TotalChunks:  req.TotalChunks,
		OutputPrefix: fmt.Sprintf("lessons/%s", req.LessonId),
	}

	// conn := getRabbitMQConnection()
	// err := publishRecordingMergeMessage(ctx, conn, message)
	// if err != nil {
	//     return nil, err
	// }

	return &CompleteRecordingResponse{
		JobId:   jobId,
		Status:  "PENDING",
		Message: "Recording merge job has been queued",
	}, nil
}

/*
MinIO Structure:
================

edtech-content/
├── recordings/                    # Temporary chunks from frontend
│   └── {lesson-id}/
│       ├── chunk-001.webm
│       ├── chunk-002.webm
│       ├── chunk-003.webm
│       └── ...
│
└── lessons/                       # Final merged videos
    └── {lesson-id}/
        └── final.mp4


Database Schema:
================

-- Jobs table
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    entity_id UUID NOT NULL,           -- lesson_id
    entity_type VARCHAR(50) NOT NULL,  -- 'lesson'
    status VARCHAR(20) NOT NULL,       -- PENDING, PROCESSING, COMPLETED, FAILED
    job_type VARCHAR(50) NOT NULL,     -- 'recording_merge'
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Lessons table (existing)
CREATE TABLE lessons (
    id UUID PRIMARY KEY,
    video_url VARCHAR(500),            -- Updated after merge: 'lessons/{id}/final.mp4'
    ... other fields ...
);

-- Recordings table (optional, for tracking chunks)
CREATE TABLE recordings (
    id UUID PRIMARY KEY,
    lesson_id UUID NOT NULL,
    chunk_number INT NOT NULL,
    chunk_path VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (lesson_id) REFERENCES lessons(id)
);


RabbitMQ Configuration:
=======================

Exchange: recording_exchange
Type: topic
Durable: true

Queue: recording_merge_queue
Routing Key: recording.merge.request
Durable: true
Arguments:
  - x-dead-letter-exchange: recording_exchange_dlx
  - x-dead-letter-routing-key: dlq.recording.merge.request

Dead Letter Queue: recording_merge_queue_dlq
Routing Key: dlq.recording.merge.request


Frontend Flow:
==============

1. Start Recording
   - MediaRecorder starts
   - Create lesson record in DB

2. During Recording (every 30 seconds)
   - Generate chunk
   - Upload to: recordings/{lesson-id}/chunk-{number}.webm
   - POST /api/chunks/upload

3. Complete Recording
   - Stop MediaRecorder
   - POST /api/recordings/complete
     {
       "lessonId": "uuid",
       "totalChunks": 10
     }
   - Backend creates job and publishes to queue
   - Return jobId to frontend

4. Poll Status
   - GET /api/jobs/{jobId}
   - Returns: PENDING -> PROCESSING -> COMPLETED/FAILED

5. On COMPLETED
   - Lesson.video_url is updated
   - Frontend can now play the merged video


Worker Processing:
==================

1. Listen to recording_merge_queue
2. Receive message with jobId, lessonId, chunkFolder, totalChunks
3. Update job status: PENDING -> PROCESSING
4. Download all chunks from MinIO
5. Sort chunks by filename
6. Merge with FFmpeg:
   - Input: WebM chunks
   - Output: MP4 (H.264 + AAC)
7. Upload final.mp4 to lessons/{lesson-id}/
8. Update lesson.video_url
9. Update job status: PROCESSING -> COMPLETED
10. Cleanup temp files
*/

