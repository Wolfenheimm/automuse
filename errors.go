package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// ErrorType represents different types of errors that can occur
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "VALIDATION"
	ErrorTypeYouTube    ErrorType = "YOUTUBE"
	ErrorTypeAudio      ErrorType = "AUDIO"
	ErrorTypeDiscord    ErrorType = "DISCORD"
	ErrorTypeNetwork    ErrorType = "NETWORK"
	ErrorTypeFileSystem ErrorType = "FILESYSTEM"
	ErrorTypeQueue      ErrorType = "QUEUE"
	ErrorTypeVoice      ErrorType = "VOICE"
	ErrorTypePermission ErrorType = "PERMISSION"
	ErrorTypeRateLimit  ErrorType = "RATE_LIMIT"
	ErrorTypeInternal   ErrorType = "INTERNAL"
)

// BotError represents a structured error with context
type BotError struct {
	Type        ErrorType
	Message     string
	UserMessage string
	Cause       error
	Context     map[string]interface{}
}

// Error implements the error interface
func (e *BotError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *BotError) Unwrap() error {
	return e.Cause
}

// NewBotError creates a new BotError
func NewBotError(errorType ErrorType, message, userMessage string, cause error) *BotError {
	return &BotError{
		Type:        errorType,
		Message:     message,
		UserMessage: userMessage,
		Cause:       cause,
		Context:     make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *BotError) WithContext(key string, value interface{}) *BotError {
	e.Context[key] = value
	return e
}

// ErrorHandler handles errors consistently across the bot
type ErrorHandler struct {
	session *discordgo.Session
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(session *discordgo.Session) *ErrorHandler {
	return &ErrorHandler{session: session}
}

// Handle processes an error and sends appropriate response to Discord
func (eh *ErrorHandler) Handle(err error, channelID string) {
	if err == nil {
		return
	}

	var botErr *BotError
	var ok bool

	// Try to cast to BotError, otherwise create a generic one
	if botErr, ok = err.(*BotError); !ok {
		botErr = NewBotError(ErrorTypeInternal, err.Error(),
			"An unexpected error occurred. Please try again.", err)
	}

	// Log the error with full context
	eh.logError(botErr)

	// Send user-friendly message to Discord
	userMessage := eh.getUserMessage(botErr)
	if channelID != "" && eh.session != nil {
		_, discordErr := eh.session.ChannelMessageSend(channelID, userMessage)
		if discordErr != nil {
			log.Printf("ERROR: Failed to send error message to Discord: %v", discordErr)
		}
	}
}

// logError logs the error with appropriate level and context
func (eh *ErrorHandler) logError(err *BotError) {
	contextStr := ""
	if len(err.Context) > 0 {
		contextStr = fmt.Sprintf(" Context: %+v", err.Context)
	}

	switch err.Type {
	case ErrorTypeValidation, ErrorTypePermission:
		log.Printf("WARN: %s%s", err.Error(), contextStr)
	case ErrorTypeNetwork, ErrorTypeRateLimit:
		log.Printf("INFO: %s%s", err.Error(), contextStr)
	default:
		log.Printf("ERROR: %s%s", err.Error(), contextStr)
	}
}

// getUserMessage returns an appropriate user-facing error message
func (eh *ErrorHandler) getUserMessage(err *BotError) string {
	if err.UserMessage != "" {
		return fmt.Sprintf("**[Muse]** %s", err.UserMessage)
	}

	// Fallback messages based on error type
	switch err.Type {
	case ErrorTypeValidation:
		return "**[Muse]** Invalid command format. Use `play help` for usage information."
	case ErrorTypeYouTube:
		return "**[Muse]** YouTube error occurred. The video might be unavailable or private."
	case ErrorTypeAudio:
		return "**[Muse]** Audio processing failed. Please try again with a different video."
	case ErrorTypeDiscord:
		return "**[Muse]** Discord connection issue. Please check bot permissions."
	case ErrorTypeNetwork:
		return "**[Muse]** Network error occurred. Please try again in a moment."
	case ErrorTypeFileSystem:
		return "**[Muse]** File system error. Please contact the bot administrator."
	case ErrorTypeQueue:
		return "**[Muse]** Queue operation failed. Please try again."
	case ErrorTypeVoice:
		return "**[Muse]** Voice channel error. Make sure I have permission to join and speak."
	case ErrorTypePermission:
		return "**[Muse]** Insufficient permissions. Please check bot permissions."
	case ErrorTypeRateLimit:
		return "**[Muse]** Rate limited. Please wait a moment before trying again."
	default:
		return "**[Muse]** An unexpected error occurred. Please try again."
	}
}

// Specific error constructors for common scenarios
func NewValidationError(message string, cause error) *BotError {
	return NewBotError(ErrorTypeValidation, message, message, cause)
}

func NewYouTubeError(message, userMessage string, cause error) *BotError {
	return NewBotError(ErrorTypeYouTube, message, userMessage, cause)
}

func NewAudioError(message, userMessage string, cause error) *BotError {
	return NewBotError(ErrorTypeAudio, message, userMessage, cause)
}

func NewDiscordError(message, userMessage string, cause error) *BotError {
	return NewBotError(ErrorTypeDiscord, message, userMessage, cause)
}

func NewVoiceError(message, userMessage string, cause error) *BotError {
	return NewBotError(ErrorTypeVoice, message, userMessage, cause)
}

func NewNetworkError(message, userMessage string, cause error) *BotError {
	return NewBotError(ErrorTypeNetwork, message, userMessage, cause)
}

func NewQueueError(message, userMessage string, cause error) *BotError {
	return NewBotError(ErrorTypeQueue, message, userMessage, cause)
}

// Recovery function for goroutines
func RecoverWithErrorHandler(errorHandler *ErrorHandler, channelID string) {
	if r := recover(); r != nil {
		var err error
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("panic recovered: %v", r)
		}

		botErr := NewBotError(ErrorTypeInternal,
			fmt.Sprintf("Panic recovered: %v", r),
			"An internal error occurred. The operation has been safely stopped.", err)

		errorHandler.Handle(botErr, channelID)
	}
}
