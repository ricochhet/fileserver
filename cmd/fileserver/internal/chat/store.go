package chat

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	chatdb "github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

const maxMessagesPerChannel = 200

var ErrNotSubscribed = errors.New("chat: not subscribed to channel")

type Message struct {
	ID          string    `json:"id"`
	ChannelCode string    `json:"channelCode"`
	ChannelName string    `json:"channelName"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Body        string    `json:"body"`
	Timestamp   time.Time `json:"timestamp"`
}

type Channel struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Store struct {
	mu       sync.RWMutex
	channels map[string]*Channel
	messages map[string][]*Message
	subs     map[string]map[string]bool
	broker   *broker
	seq      atomic.Uint64
	db       *chatdb.DB
}

// NewStore returns a ready-to-use Store; pass a non-nil DB to enable persistence.
func NewStore(database *chatdb.DB) *Store {
	s := &Store{
		channels: make(map[string]*Channel),
		messages: make(map[string][]*Message),
		subs:     make(map[string]map[string]bool),
		broker:   newBroker(),
		db:       database,
	}

	if database != nil {
		// Seed the counter from the DB so IDs never go backwards after a restart.
		if maxID, err := database.MaxMessageID(context.Background()); err == nil && maxID > 0 {
			s.seq.Store(maxID)
		}
	}

	return s
}

// SeedChannel creates a channel if it does not already exist.
func (s *Store) SeedChannel(code, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.channels[code]; ok {
		return
	}

	if name == "" {
		name = "#" + code
	}

	s.channels[code] = &Channel{Code: code, Name: name}
}

// JoinChannel subscribes username to the channel, creating it if necessary.
func (s *Store) JoinChannel(username, code, name string) *Channel {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.channels[code]
	if !ok {
		n := name
		if n == "" {
			n = "#" + code
		}

		ch = &Channel{Code: code, Name: n}
		s.channels[code] = ch
	}

	if s.subs[username] == nil {
		s.subs[username] = make(map[string]bool)
	}

	s.subs[username][code] = true

	return ch
}

// LeaveChannel removes username's subscription to the channel.
func (s *Store) LeaveChannel(username, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subs[username] != nil {
		delete(s.subs[username], code)
	}
}

// Subscriptions returns all channels username is subscribed to.
func (s *Store) Subscriptions(username string) []*Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*Channel

	for code := range s.subs[username] {
		if ch, ok := s.channels[code]; ok {
			out = append(out, ch)
		}
	}

	return out
}

// AllChannels returns every channel known to the store, regardless of subscriptions.
func (s *Store) AllChannels() []*Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		out = append(out, ch)
	}

	return out
}

// IsSubscribed reports whether username holds a subscription to code.
func (s *Store) IsSubscribed(username, code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.subs[username][code]
}

// Post appends and broadcasts a message, returning ErrNotSubscribed if the user is not subscribed.
func (s *Store) Post(
	ctx context.Context,
	username, displayName, code, body string,
) (*Message, error) {
	s.mu.Lock()

	if !s.subs[username][code] {
		s.mu.Unlock()
		return nil, ErrNotSubscribed
	}

	ch := s.channels[code]
	msg := &Message{
		ID:          s.nextID(),
		ChannelCode: code,
		ChannelName: ch.Name,
		Username:    username,
		DisplayName: displayName,
		Body:        body,
		Timestamp:   time.Now().UTC(),
	}

	msgs := s.messages[code]
	msgs = append(msgs, msg)

	if len(msgs) > maxMessagesPerChannel {
		msgs = msgs[len(msgs)-maxMessagesPerChannel:]
	}

	s.messages[code] = msgs

	s.mu.Unlock()

	// Persist and publish outside the lock so disk latency cannot stall other writers or the broker.
	if s.db != nil {
		if err := s.db.SaveMessage(ctx, toDBMessage(msg)); err != nil {
			logutil.Errorf(logutil.Get(), "chat.Store.Post: persist message %s: %v\n", msg.ID, err)
		}
	}

	s.broker.publish(msg)

	return msg, nil
}

// Messages returns history for the channel; reads from the DB when available for restart persistence.
func (s *Store) Messages(ctx context.Context, username, code string) ([]*Message, error) {
	s.mu.RLock()
	subscribed := s.subs[username][code]
	s.mu.RUnlock()

	if !subscribed {
		return nil, ErrNotSubscribed
	}

	if s.db != nil {
		dbMsgs, err := s.db.GetMessages(ctx, code, maxMessagesPerChannel)
		if err != nil {
			return nil, err
		}

		out := make([]*Message, len(dbMsgs))
		for i, m := range dbMsgs {
			out[i] = fromDBMessage(m)
		}

		return out, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	src := s.messages[code]
	out := make([]*Message, len(src))
	copy(out, src)

	return out, nil
}

// Subscribe registers an SSE listener for username, returning the channel and a cancel func.
func (s *Store) Subscribe(username string) (<-chan *Message, func()) {
	return s.broker.subscribe(username)
}

// nextID returns the next monotonically increasing message ID as a string.
func (s *Store) nextID() string {
	return strconv.FormatUint(s.seq.Add(1), 10)
}

// toDBMessage converts a Message to a chatdb.Message.
func toDBMessage(m *Message) *chatdb.Message {
	return &chatdb.Message{
		ID:          m.ID,
		ChannelCode: m.ChannelCode,
		ChannelName: m.ChannelName,
		Username:    m.Username,
		DisplayName: m.DisplayName,
		Body:        m.Body,
		Timestamp:   m.Timestamp,
	}
}

// fromDBMessage converts a chatdb.Message to a Message.
func fromDBMessage(m *chatdb.Message) *Message {
	return &Message{
		ID:          m.ID,
		ChannelCode: m.ChannelCode,
		ChannelName: m.ChannelName,
		Username:    m.Username,
		DisplayName: m.DisplayName,
		Body:        m.Body,
		Timestamp:   m.Timestamp,
	}
}

// broker fans messages out to all registered SSE listeners.
type broker struct {
	mu      sync.RWMutex
	clients map[string][]chan *Message
}

// newBroker returns a new broker.
func newBroker() *broker {
	return &broker{clients: make(map[string][]chan *Message)}
}

// subscribe registers a buffered listener channel and returns it with a cancel func.
func (b *broker) subscribe(username string) (<-chan *Message, func()) {
	ch := make(chan *Message, 64)

	b.mu.Lock()
	b.clients[username] = append(b.clients[username], ch)
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		list := b.clients[username]
		for i, c := range list {
			if c == ch {
				b.clients[username] = append(list[:i], list[i+1:]...)
				break
			}
		}

		close(ch)
	}

	return ch, cancel
}

// publish delivers msg to all listeners, dropping for slow consumers who will catch up via Messages().
func (b *broker) publish(msg *Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, list := range b.clients {
		for _, ch := range list {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}
