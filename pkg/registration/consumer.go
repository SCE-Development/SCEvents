package registration

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		CommitInterval: 0,
	})

	return &Consumer{reader: r}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("consumer fetch error: %v", err)
			continue
		}

		if err := ProcessKafkaMessage(msg.Value); err != nil {
			log.Printf("consumer process error: %v", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("consumer commit error: %v", err)
		}
	}
}

// Close releases the Kafka reader.
func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func ProcessKafkaMessage(raw []byte) error {
	var msg models.KafkaRegistrationMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return err
	}

	if err := validateKafkaMessage(msg); err != nil {
		return err
	}

	req, err := db.GetRegistrationByID(msg.RequestID)
	if err != nil {
		return err
	}

	if req.Status != models.StatusPending {
		return nil
	}

	_, err = db.GetEventByID(msg.EventID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return db.MarkRegistrationRejected(msg.RequestID, models.ReasonEventNotFound)
		}
		return err
	}

	duplicate, err := db.HasAcceptedRegistration(msg.EventID, msg.UserID)
	if err != nil {
		return err
	}
	if duplicate {
		return db.MarkRegistrationRejected(msg.RequestID, models.ReasonDuplicateUser)
	}

	ok, err := db.TryTakeEventSeat(msg.EventID)
	if err != nil {
		return err
	}
	if !ok {
		return db.MarkRegistrationRejected(msg.RequestID, models.ReasonCapacityFull)
	}

	if err := db.MarkRegistrationAccepted(msg.RequestID); err != nil {
		_ = db.ReleaseEventSeat(msg.EventID)
		return err
	}

	return nil
}

func validateKafkaMessage(msg models.KafkaRegistrationMessage) error {
	if msg.RequestID == "" {
		return errors.New("missing request_id")
	}
	if msg.EventID == "" {
		return errors.New("missing event_id")
	}
	if msg.UserID == "" {
		return errors.New("missing user_id")
	}
	return nil
}