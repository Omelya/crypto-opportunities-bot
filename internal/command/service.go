package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Command types
const (
	CommandTriggerScraper           = "trigger_scraper"
	CommandTriggerAllScrapers       = "trigger_all_scrapers"
	CommandClearCache               = "clear_cache"
	CommandRestartDispatcher        = "restart_dispatcher"
	CommandGetScraperStatus         = "get_scraper_status"
	CommandGetExchangeStatus        = "get_exchange_status"
	CommandTriggerDeFiScrape        = "trigger_defi_scrape"
	CommandGetArbitrageDetectorInfo = "get_arbitrage_info"
)

// CommandMessage represents a command to be executed
type CommandMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// CommandResponse represents a response to a command
type CommandResponse struct {
	ID        string                 `json:"id"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Service manages command execution via Redis Pub/Sub
type Service struct {
	redis      *redis.Client
	commandCh  chan *CommandMessage
	responseCh chan *CommandResponse
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewService creates a new command service
func NewService(redisClient *redis.Client) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		redis:      redisClient,
		commandCh:  make(chan *CommandMessage, 100),
		responseCh: make(chan *CommandResponse, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts listening for commands
func (s *Service) Start() error {
	if s.redis == nil {
		log.Printf("⚠️  Command service: Redis not available, running in standalone mode")
		return nil
	}

	// Subscribe to command channel
	pubsub := s.redis.Subscribe(s.ctx, "bot:commands")
	defer pubsub.Close()

	log.Printf("✅ Command service started, listening for commands")

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				msg, err := pubsub.ReceiveMessage(s.ctx)
				if err != nil {
					if s.ctx.Err() != nil {
						return
					}
					log.Printf("Error receiving message: %v", err)
					continue
				}

				var cmd CommandMessage
				if err := json.Unmarshal([]byte(msg.Payload), &cmd); err != nil {
					log.Printf("Error unmarshaling command: %v", err)
					continue
				}

				s.commandCh <- &cmd
			}
		}
	}()

	return nil
}

// Stop stops the command service
func (s *Service) Stop() {
	s.cancel()
	close(s.commandCh)
	close(s.responseCh)
}

// SendCommand sends a command and waits for response with timeout
func (s *Service) SendCommand(ctx context.Context, cmdType string, payload map[string]interface{}) (*CommandResponse, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	cmd := &CommandMessage{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      cmdType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Publish command
	if err := s.redis.Publish(ctx, "bot:commands", data).Err(); err != nil {
		return nil, fmt.Errorf("failed to publish command: %w", err)
	}

	// Wait for response with timeout
	responseCh := make(chan *CommandResponse, 1)
	defer close(responseCh)

	// Subscribe to response channel
	pubsub := s.redis.Subscribe(ctx, fmt.Sprintf("bot:responses:%s", cmd.ID))
	defer pubsub.Close()

	go func() {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			return
		}

		var resp CommandResponse
		if err := json.Unmarshal([]byte(msg.Payload), &resp); err != nil {
			log.Printf("Error unmarshaling response: %v", err)
			return
		}

		responseCh <- &resp
	}()

	// Wait for response or timeout
	timeout := time.After(10 * time.Second)
	select {
	case resp := <-responseCh:
		return resp, nil
	case <-timeout:
		return nil, fmt.Errorf("command timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetCommandChannel returns the command channel for processing
func (s *Service) GetCommandChannel() <-chan *CommandMessage {
	return s.commandCh
}

// SendResponse sends a response to a command
func (s *Service) SendResponse(cmdID string, success bool, data map[string]interface{}, errMsg string) error {
	if s.redis == nil {
		return nil
	}

	resp := &CommandResponse{
		ID:        cmdID,
		Success:   success,
		Data:      data,
		Error:     errMsg,
		Timestamp: time.Now(),
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Publish response to specific channel
	channel := fmt.Sprintf("bot:responses:%s", cmdID)
	if err := s.redis.Publish(s.ctx, channel, respData).Err(); err != nil {
		return fmt.Errorf("failed to publish response: %w", err)
	}

	return nil
}
