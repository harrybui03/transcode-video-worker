package handler

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"worker-transcode/dto"
	"worker-transcode/service"
)

func JobHandler(ctx context.Context, msg amqp.Delivery, deps service.Service) error {
	var job dto.JobMessage
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		return err
	}

	err := deps.Process(ctx, job)
	if err != nil {
		return err
	}

	return nil
}
