package audio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

// Manager handles audio operations
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	config   Config
}

// Config holds audio configuration
type Config struct {
	Bitrate           int
	Volume            int
	FrameRate         int
	FrameDuration     int
	CompressionLevel  int
	PacketLoss        int
	BufferedFrames    int
	EnableVBR         bool
	ConnectTimeout    time.Duration
	SpeakingTimeout   time.Duration
}

// Session represents an audio session for a guild
type Session struct {
	mu            sync.RWMutex
	guildID       string
	channelID     string
	voice         *discordgo.VoiceConnection
	encoder       *dca.EncodeSession
	stream        *dca.StreamingSession
	currentTrack  *Track
	state         State
	volume        int
	paused        bool
	speaking      bool
	lastActivity  time.Time
	listeners     []chan StateChange
}

// Track represents an audio track
type Track struct {
	ID       string
	Title    string
	URL      string
	Duration time.Duration
	Metadata map[string]interface{}
}

// State represents the playback state
type State string

const (
	StateIdle    State = "idle"
	StatePlaying State = "playing"
	StatePaused  State = "paused"
	StateStopped State = "stopped"
	StateError   State = "error"
)

// StateChange represents a state change event
type StateChange struct {
	GuildID   string
	OldState  State
	NewState  State
	Track     *Track
	Timestamp time.Time
	Error     error
}

// NewManager creates a new audio manager
func NewManager(config Config) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		config:   config,
	}
}

// GetSession returns an audio session for a guild
func (m *Manager) GetSession(guildID string) (*Session, error) {
	m.mu.RLock()
	session, exists := m.sessions[guildID]
	m.mu.RUnlock()
	
	if exists {
		return session, nil
	}
	
	// Create new session
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check after acquiring write lock
	if session, exists := m.sessions[guildID]; exists {
		return session, nil
	}
	
	session = &Session{
		guildID:      guildID,
		state:        StateIdle,
		volume:       m.config.Volume,
		lastActivity: time.Now(),
		listeners:    make([]chan StateChange, 0),
	}
	
	m.sessions[guildID] = session
	return session, nil
}

// JoinChannel joins a voice channel
func (m *Manager) JoinChannel(ctx context.Context, session *discordgo.Session, guildID, channelID string) error {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	// If already connected to the same channel, return
	if audioSession.voice != nil && audioSession.channelID == channelID {
		return nil
	}
	
	// Disconnect from current channel if connected
	if audioSession.voice != nil {
		audioSession.voice.Disconnect()
		audioSession.voice = nil
	}
	
	// Create context with timeout
	connectCtx, cancel := context.WithTimeout(ctx, m.config.ConnectTimeout)
	defer cancel()
	
	// Join voice channel
	voice, err := session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %w", err)
	}
	
	// Wait for connection to be ready
	select {
	case <-connectCtx.Done():
		voice.Disconnect()
		return fmt.Errorf("voice connection timeout")
	case <-time.After(100 * time.Millisecond):
		// Check if connection is ready
		ready := false
		for i := 0; i < 50; i++ { // 5 second timeout
			if voice.Ready {
				ready = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		
		if !ready {
			voice.Disconnect()
			return fmt.Errorf("voice connection failed to become ready")
		}
	}
	
	audioSession.voice = voice
	audioSession.channelID = channelID
	audioSession.lastActivity = time.Now()
	
	return nil
}

// LeaveChannel leaves the current voice channel
func (m *Manager) LeaveChannel(guildID string) error {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	if audioSession.voice != nil {
		audioSession.voice.Disconnect()
		audioSession.voice = nil
	}
	
	audioSession.channelID = ""
	audioSession.setState(StateIdle)
	
	return nil
}

// Play plays a track
func (m *Manager) Play(ctx context.Context, guildID string, track *Track) error {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	if audioSession.voice == nil {
		return fmt.Errorf("not connected to a voice channel")
	}
	
	// Stop current playback if any
	if audioSession.encoder != nil {
		audioSession.encoder.Cleanup()
		audioSession.encoder = nil
	}
	
	if audioSession.stream != nil {
		audioSession.stream.Finished()
		audioSession.stream = nil
	}
	
	// Create DCA encoder options
	options := dca.StdEncodeOptions
	options.RawOutput = false
	options.Bitrate = m.config.Bitrate
	options.Application = dca.AudioApplicationAudio
	options.Volume = audioSession.volume
	options.CompressionLevel = m.config.CompressionLevel
	options.FrameRate = m.config.FrameRate
	options.FrameDuration = m.config.FrameDuration
	options.PacketLoss = m.config.PacketLoss
	options.VBR = m.config.EnableVBR
	options.BufferedFrames = m.config.BufferedFrames
	
	// Create encoder session
	encoder, err := dca.EncodeFile(track.URL, options)
	if err != nil {
		audioSession.setState(StateError)
		return fmt.Errorf("failed to create encoder: %w", err)
	}
	
	audioSession.encoder = encoder
	audioSession.currentTrack = track
	audioSession.setState(StatePlaying)
	
	// Start playback in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				audioSession.setState(StateError)
			}
		}()
		
		// Create streaming session
		stream := dca.NewStream(encoder, audioSession.voice, nil)
		audioSession.stream = stream
		
		// Set speaking
		audioSession.voice.Speaking(true)
		audioSession.speaking = true
		
		// Wait for playback to finish
		go func() {
			// Monitor stream for completion
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
					if audioSession.paused {
						continue
					}
					// Check if encoder is still running
					if encoder.Running() {
						continue
					}
					// Encoder finished
					audioSession.mu.Lock()
					audioSession.voice.Speaking(false)
					audioSession.speaking = false
					audioSession.setState(StateIdle)
					audioSession.currentTrack = nil
					audioSession.mu.Unlock()
					return
				}
			}
		}()
	}()
	
	return nil
}

// Pause pauses playback
func (m *Manager) Pause(guildID string) error {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	if audioSession.state != StatePlaying {
		return fmt.Errorf("not playing")
	}
	
	audioSession.paused = true
	audioSession.setState(StatePaused)
	
	if audioSession.voice != nil {
		audioSession.voice.Speaking(false)
		audioSession.speaking = false
	}
	
	return nil
}

// Resume resumes playback
func (m *Manager) Resume(guildID string) error {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	if audioSession.state != StatePaused {
		return fmt.Errorf("not paused")
	}
	
	audioSession.paused = false
	audioSession.setState(StatePlaying)
	
	if audioSession.voice != nil {
		audioSession.voice.Speaking(true)
		audioSession.speaking = true
	}
	
	return nil
}

// Stop stops playback
func (m *Manager) Stop(guildID string) error {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	if audioSession.encoder != nil {
		audioSession.encoder.Cleanup()
		audioSession.encoder = nil
	}
	
	if audioSession.stream != nil {
		audioSession.stream.Finished()
		audioSession.stream = nil
	}
	
	if audioSession.voice != nil {
		audioSession.voice.Speaking(false)
		audioSession.speaking = false
	}
	
	audioSession.paused = false
	audioSession.currentTrack = nil
	audioSession.setState(StateStopped)
	
	return nil
}

// SetVolume sets the volume for a guild
func (m *Manager) SetVolume(guildID string, volume int) error {
	if volume < 0 || volume > 1024 {
		return fmt.Errorf("volume must be between 0 and 1024")
	}
	
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	audioSession.volume = volume
	
	return nil
}

// GetState returns the current state of an audio session
func (m *Manager) GetState(guildID string) (State, *Track, error) {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return StateIdle, nil, fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.RLock()
	defer audioSession.mu.RUnlock()
	
	return audioSession.state, audioSession.currentTrack, nil
}

// IsConnected returns true if connected to a voice channel
func (m *Manager) IsConnected(guildID string) bool {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return false
	}
	
	audioSession.mu.RLock()
	defer audioSession.mu.RUnlock()
	
	return audioSession.voice != nil && audioSession.voice.Ready
}

// GetCurrentTrack returns the currently playing track
func (m *Manager) GetCurrentTrack(guildID string) *Track {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return nil
	}
	
	audioSession.mu.RLock()
	defer audioSession.mu.RUnlock()
	
	return audioSession.currentTrack
}

// Subscribe subscribes to state changes
func (m *Manager) Subscribe(guildID string) (<-chan StateChange, error) {
	audioSession, err := m.GetSession(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audio session: %w", err)
	}
	
	audioSession.mu.Lock()
	defer audioSession.mu.Unlock()
	
	listener := make(chan StateChange, 10)
	audioSession.listeners = append(audioSession.listeners, listener)
	
	return listener, nil
}

// CleanupInactiveSessions removes inactive sessions
func (m *Manager) CleanupInactiveSessions(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	
	for guildID, session := range m.sessions {
		session.mu.RLock()
		lastActivity := session.lastActivity
		isActive := session.voice != nil && session.state == StatePlaying
		session.mu.RUnlock()
		
		if !isActive && lastActivity.Before(cutoff) {
			// Clean up session
			session.mu.Lock()
			if session.voice != nil {
				session.voice.Disconnect()
			}
			if session.encoder != nil {
				session.encoder.Cleanup()
			}
			if session.stream != nil {
				session.stream.Finished()
			}
			session.mu.Unlock()
			
			delete(m.sessions, guildID)
		}
	}
}

// GetActiveSessions returns the number of active sessions
func (m *Manager) GetActiveSessions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	active := 0
	for _, session := range m.sessions {
		session.mu.RLock()
		if session.voice != nil && session.state == StatePlaying {
			active++
		}
		session.mu.RUnlock()
	}
	
	return active
}

// setState sets the state and notifies listeners
func (s *Session) setState(newState State) {
	oldState := s.state
	s.state = newState
	s.lastActivity = time.Now()
	
	// Notify listeners
	change := StateChange{
		GuildID:   s.guildID,
		OldState:  oldState,
		NewState:  newState,
		Track:     s.currentTrack,
		Timestamp: time.Now(),
	}
	
	for _, listener := range s.listeners {
		select {
		case listener <- change:
		default:
			// Listener buffer full, skip
		}
	}
}

// Shutdown shuts down the audio manager
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, session := range m.sessions {
		session.mu.Lock()
		if session.voice != nil {
			session.voice.Disconnect()
		}
		if session.encoder != nil {
			session.encoder.Cleanup()
		}
		if session.stream != nil {
			session.stream.Finished()
		}
		session.mu.Unlock()
	}
	
	m.sessions = make(map[string]*Session)
	return nil
}