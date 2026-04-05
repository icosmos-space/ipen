package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/icosmos-space/ipen/core/utils"
)

// SchedulerConfig 表示scheduler configuration。
type SchedulerConfig struct {
	PipelineConfig
	RadarCron              string
	WriteCron              string
	MaxConcurrentBooks     int
	ChaptersPerCycle       int
	RetryDelayMs           int
	CooldownAfterChapterMs int
	MaxChaptersPerDay      int
	QualityGates           *models.QualityGates
	Detection              *models.DetectionConfig
	OnChapterComplete      func(bookID string, chapter int, status string)
	OnError                func(bookID string, err error)
	OnPause                func(bookID string, reason string)
}

// ScheduledTask 表示a scheduled task。
type ScheduledTask struct {
	Name       string
	IntervalMs int
	Timer      *time.Ticker
}

// Scheduler 表示the scheduler。
type Scheduler struct {
	pipeline           *PipelineRunner
	stateManager       *state.StateManager
	config             SchedulerConfig
	tasks              []ScheduledTask
	running            bool
	writeCycleInFlight bool
	radarScanInFlight  bool

	// Quality gate tracking
	consecutiveFailures map[string]int
	pausedBooks         map[string]bool
	failureDimensions   map[string]map[string]int
	dailyChapterCount   map[string]int

	logger utils.Logger
	mu     sync.Mutex
}

// NewScheduler 创建新的scheduler。
func NewScheduler(config SchedulerConfig) *Scheduler {
	return &Scheduler{
		pipeline:            NewPipelineRunner(config.PipelineConfig),
		stateManager:        state.NewStateManager(config.ProjectRoot),
		config:              config,
		consecutiveFailures: make(map[string]int),
		pausedBooks:         make(map[string]bool),
		failureDimensions:   make(map[string]map[string]int),
		dailyChapterCount:   make(map[string]int),
		logger:              config.Logger,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	if s.running {
		return nil
	}
	s.running = true

	// Run write cycle immediately
	if err := s.triggerWriteCycle(ctx); err != nil {
		s.config.OnError("scheduler", err)
	}

	// Schedule recurring write cycle
	writeInterval := CronToMs(s.config.WriteCron)
	writeTicker := time.NewTicker(time.Duration(writeInterval) * time.Millisecond)
	s.tasks = append(s.tasks, ScheduledTask{
		Name:       "write-cycle",
		IntervalMs: writeInterval,
		Timer:      writeTicker,
	})

	go func() {
		for range writeTicker.C {
			if err := s.triggerWriteCycle(ctx); err != nil {
				s.config.OnError("scheduler", err)
			}
		}
	}()

	// Schedule radar scan
	radarInterval := CronToMs(s.config.RadarCron)
	radarTicker := time.NewTicker(time.Duration(radarInterval) * time.Millisecond)
	s.tasks = append(s.tasks, ScheduledTask{
		Name:       "radar-scan",
		IntervalMs: radarInterval,
		Timer:      radarTicker,
	})

	go func() {
		for range radarTicker.C {
			if err := s.triggerRadarScan(ctx); err != nil {
				s.config.OnError("scheduler", err)
			}
		}
	}()

	s.logger.Info("Scheduler started", nil)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	for _, task := range s.tasks {
		if task.Timer != nil {
			task.Timer.Stop()
		}
	}
	s.tasks = nil
	s.logger.Info("Scheduler stopped", nil)
}

func (s *Scheduler) triggerWriteCycle(ctx context.Context) error {
	s.mu.Lock()
	if s.writeCycleInFlight {
		s.mu.Unlock()
		return nil
	}
	s.writeCycleInFlight = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.writeCycleInFlight = false
		s.mu.Unlock()
	}()

	books, err := s.stateManager.ListBooks()
	if err != nil {
		return fmt.Errorf("failed to list books: %w", err)
	}

	for _, bookID := range books {
		if s.pausedBooks[bookID] {
			continue
		}

		if err := s.processBook(ctx, bookID); err != nil {
			s.config.OnError(bookID, err)
		}
	}

	return nil
}

func (s *Scheduler) triggerRadarScan(ctx context.Context) error {
	s.mu.Lock()
	if s.radarScanInFlight {
		s.mu.Unlock()
		return nil
	}
	s.radarScanInFlight = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.radarScanInFlight = false
		s.mu.Unlock()
	}()

	// Radar scan implementation
	// Would call RadarAgent.ScanRankings for each book
	s.logger.Info("Radar scan completed", nil)
	return nil
}

func (s *Scheduler) processBook(ctx context.Context, bookID string) error {
	bookConfig, err := s.stateManager.LoadBookConfig(bookID)
	if err != nil {
		return err
	}

	if bookConfig.Status == "paused" || bookConfig.Status == "completed" || bookConfig.Status == "dropped" {
		return nil
	}

	// Check daily limit
	today := time.Now().Format("2006-01-02")
	if s.dailyChapterCount[today] >= s.config.MaxChaptersPerDay {
		s.logger.Warn("Daily chapter limit reached", map[string]any{
			"limit": s.config.MaxChaptersPerDay,
		})
		return nil
	}

	// Get next chapter number
	nextChapter, err := s.stateManager.GetNextChapterNumber(bookID)
	if err != nil {
		return err
	}

	// Run chapter pipeline
	result, err := s.pipeline.RunChapterPipeline(ctx, bookID, nextChapter)
	if err != nil {
		s.consecutiveFailures[bookID]++
		if s.config.QualityGates != nil && s.consecutiveFailures[bookID] >= s.config.QualityGates.PauseAfterConsecutiveFailures {
			s.pausedBooks[bookID] = true
			s.config.OnPause(bookID, fmt.Sprintf("consecutive failures: %d", s.consecutiveFailures[bookID]))
		}
		return err
	}

	// Reset failure count on success
	s.consecutiveFailures[bookID] = 0

	// Update daily count
	s.dailyChapterCount[today]++

	// Save chapter index
	chapterIndex, _ := s.stateManager.LoadChapterIndex(bookID)
	chapterIndex = append(chapterIndex, models.ChapterMeta{
		Number:    nextChapter,
		Title:     result.Title,
		Status:    models.StatusDrafted,
		WordCount: result.WordCount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err := s.stateManager.SaveChapterIndex(bookID, chapterIndex); err != nil {
		return err
	}

	// Send notification
	if s.config.OnChapterComplete != nil {
		s.config.OnChapterComplete(bookID, nextChapter, result.Status)
	}

	// Cooldown
	time.Sleep(time.Duration(s.config.CooldownAfterChapterMs) * time.Millisecond)

	return nil
}

// CronToMs converts cron expression to milliseconds
func CronToMs(cronExpr string) int {
	// Simple implementation - would need proper cron parser
	// For "*/15 * * * *" -> 15 minutes = 900000 ms
	// For "0 */6 * * *" -> 6 hours = 21600000 ms
	if cronExpr == "*/15 * * * *" {
		return 15 * 60 * 1000
	}
	if cronExpr == "0 */6 * * *" {
		return 6 * 60 * 60 * 1000
	}
	return 15 * 60 * 1000 // default 15 minutes
}

// IsBookPaused 检查if a book is paused。
func (s *Scheduler) IsBookPaused(bookID string) bool {
	return s.pausedBooks[bookID]
}

// PauseBook pauses a book
func (s *Scheduler) PauseBook(bookID string, reason string) {
	s.pausedBooks[bookID] = true
	if s.config.OnPause != nil {
		s.config.OnPause(bookID, reason)
	}
}

// ResumeBook resumes a book
func (s *Scheduler) ResumeBook(bookID string) {
	delete(s.pausedBooks, bookID)
}
