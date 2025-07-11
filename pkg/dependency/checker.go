package dependency

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Dependency represents a system dependency
type Dependency struct {
	Name        string
	Command     string
	Args        []string
	Required    bool
	Description string
	MinVersion  string
	InstallCmd  string
}

// CheckResult represents the result of a dependency check
type CheckResult struct {
	Dependency    Dependency
	Available     bool
	Version       string
	Error         error
	InstallHint   string
}

// Checker handles dependency checking
type Checker struct {
	timeout time.Duration
}

// NewChecker creates a new dependency checker
func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		timeout: timeout,
	}
}

// CheckAll checks all dependencies and returns results
func (c *Checker) CheckAll(ctx context.Context, deps []Dependency) ([]CheckResult, error) {
	results := make([]CheckResult, 0, len(deps))
	
	for _, dep := range deps {
		result := c.checkSingle(ctx, dep)
		results = append(results, result)
	}
	
	return results, nil
}

// CheckRequired checks only required dependencies
func (c *Checker) CheckRequired(ctx context.Context, deps []Dependency) error {
	var missingRequired []string
	
	for _, dep := range deps {
		if !dep.Required {
			continue
		}
		
		result := c.checkSingle(ctx, dep)
		if !result.Available {
			missingRequired = append(missingRequired, dep.Name)
		}
	}
	
	if len(missingRequired) > 0 {
		return fmt.Errorf("missing required dependencies: %s", strings.Join(missingRequired, ", "))
	}
	
	return nil
}

// checkSingle checks a single dependency
func (c *Checker) checkSingle(ctx context.Context, dep Dependency) CheckResult {
	result := CheckResult{
		Dependency: dep,
		Available:  false,
	}
	
	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	// Execute command
	cmd := exec.CommandContext(cmdCtx, dep.Command, dep.Args...)
	output, err := cmd.Output()
	
	if err != nil {
		result.Error = err
		result.InstallHint = dep.InstallCmd
		return result
	}
	
	result.Available = true
	result.Version = strings.TrimSpace(string(output))
	
	return result
}

// GetSystemDependencies returns the list of system dependencies for AutoMuse
func GetSystemDependencies() []Dependency {
	return []Dependency{
		{
			Name:        "FFmpeg",
			Command:     "ffmpeg",
			Args:        []string{"-version"},
			Required:    true,
			Description: "Audio/video processing library required for media conversion",
			MinVersion:  "4.0.0",
			InstallCmd:  "brew install ffmpeg (macOS) | apt-get install ffmpeg (Ubuntu) | yum install ffmpeg (CentOS)",
		},
		{
			Name:        "yt-dlp",
			Command:     "yt-dlp",
			Args:        []string{"--version"},
			Required:    false,
			Description: "YouTube downloader used as fallback for restricted videos",
			MinVersion:  "2023.01.06",
			InstallCmd:  "pip install yt-dlp | brew install yt-dlp",
		},
		{
			Name:        "youtube-dl",
			Command:     "youtube-dl",
			Args:        []string{"--version"},
			Required:    false,
			Description: "Legacy YouTube downloader (fallback to yt-dlp)",
			MinVersion:  "2021.12.17",
			InstallCmd:  "pip install youtube-dl | brew install youtube-dl",
		},
		{
			Name:        "opus",
			Command:     "opusenc",
			Args:        []string{"--version"},
			Required:    false,
			Description: "Opus audio codec for high-quality audio compression",
			MinVersion:  "1.3.0",
			InstallCmd:  "brew install opus-tools | apt-get install opus-tools",
		},
	}
}

// ValidateEnvironment validates the complete environment
func ValidateEnvironment(ctx context.Context) (*EnvironmentReport, error) {
	checker := NewChecker(10 * time.Second)
	deps := GetSystemDependencies()
	
	results, err := checker.CheckAll(ctx, deps)
	if err != nil {
		return nil, fmt.Errorf("failed to check dependencies: %w", err)
	}
	
	report := &EnvironmentReport{
		CheckTime: time.Now(),
		Results:   results,
	}
	
	// Analyze results
	report.analyzeResults()
	
	return report, nil
}

// EnvironmentReport contains the results of environment validation
type EnvironmentReport struct {
	CheckTime         time.Time
	Results           []CheckResult
	RequiredMissing   []string
	OptionalMissing   []string
	RecommendedAction string
	Severity          string
}

// analyzeResults analyzes the check results and provides recommendations
func (r *EnvironmentReport) analyzeResults() {
	var requiredMissing []string
	var optionalMissing []string
	
	for _, result := range r.Results {
		if !result.Available {
			if result.Dependency.Required {
				requiredMissing = append(requiredMissing, result.Dependency.Name)
			} else {
				optionalMissing = append(optionalMissing, result.Dependency.Name)
			}
		}
	}
	
	r.RequiredMissing = requiredMissing
	r.OptionalMissing = optionalMissing
	
	// Determine severity and action
	if len(requiredMissing) > 0 {
		r.Severity = "CRITICAL"
		r.RecommendedAction = "Install required dependencies before starting the application"
	} else if len(optionalMissing) > 0 {
		r.Severity = "WARNING"
		r.RecommendedAction = "Consider installing optional dependencies for full functionality"
	} else {
		r.Severity = "OK"
		r.RecommendedAction = "All dependencies are available"
	}
}

// IsHealthy returns true if all required dependencies are available
func (r *EnvironmentReport) IsHealthy() bool {
	return len(r.RequiredMissing) == 0
}

// GetInstallCommands returns installation commands for missing dependencies
func (r *EnvironmentReport) GetInstallCommands() []string {
	var commands []string
	
	for _, result := range r.Results {
		if !result.Available && result.InstallHint != "" {
			commands = append(commands, fmt.Sprintf("# %s\n%s", result.Dependency.Description, result.InstallHint))
		}
	}
	
	return commands
}

// GenerateReport generates a human-readable report
func (r *EnvironmentReport) GenerateReport() string {
	var report strings.Builder
	
	report.WriteString("=== AutoMuse Environment Report ===\n")
	report.WriteString(fmt.Sprintf("Check Time: %s\n", r.CheckTime.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Severity: %s\n", r.Severity))
	report.WriteString(fmt.Sprintf("Recommended Action: %s\n\n", r.RecommendedAction))
	
	report.WriteString("Dependency Status:\n")
	for _, result := range r.Results {
		status := "✓ Available"
		if !result.Available {
			status = "✗ Missing"
		}
		
		required := ""
		if result.Dependency.Required {
			required = " (Required)"
		}
		
		report.WriteString(fmt.Sprintf("  %s: %s%s\n", result.Dependency.Name, status, required))
		if result.Version != "" {
			report.WriteString(fmt.Sprintf("    Version: %s\n", result.Version))
		}
		if result.Error != nil {
			report.WriteString(fmt.Sprintf("    Error: %s\n", result.Error.Error()))
		}
	}
	
	if len(r.RequiredMissing) > 0 {
		report.WriteString("\n⚠️  Required Dependencies Missing:\n")
		for _, dep := range r.RequiredMissing {
			report.WriteString(fmt.Sprintf("  - %s\n", dep))
		}
	}
	
	if len(r.OptionalMissing) > 0 {
		report.WriteString("\nOptional Dependencies Missing:\n")
		for _, dep := range r.OptionalMissing {
			report.WriteString(fmt.Sprintf("  - %s\n", dep))
		}
	}
	
	if len(r.GetInstallCommands()) > 0 {
		report.WriteString("\nInstallation Commands:\n")
		for _, cmd := range r.GetInstallCommands() {
			report.WriteString(fmt.Sprintf("%s\n\n", cmd))
		}
	}
	
	return report.String()
}

// CheckFFmpegFeatures checks specific FFmpeg features
func CheckFFmpegFeatures(ctx context.Context) (map[string]bool, error) {
	features := make(map[string]bool)
	
	// Check for specific codec support
	codecs := []string{"libopus", "libvorbis", "aac", "mp3"}
	
	for _, codec := range codecs {
		cmd := exec.CommandContext(ctx, "ffmpeg", "-codecs")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to check FFmpeg codecs: %w", err)
		}
		
		features[codec] = strings.Contains(string(output), codec)
	}
	
	return features, nil
}

// CheckYouTubeDLCapabilities checks YouTube-dl/yt-dlp capabilities
func CheckYouTubeDLCapabilities(ctx context.Context) (*YTDLCapabilities, error) {
	caps := &YTDLCapabilities{}
	
	// Check yt-dlp first
	cmd := exec.CommandContext(ctx, "yt-dlp", "--help")
	if err := cmd.Run(); err == nil {
		caps.YTDLPAvailable = true
		caps.PreferredTool = "yt-dlp"
	} else {
		// Check youtube-dl as fallback
		cmd = exec.CommandContext(ctx, "youtube-dl", "--help")
		if err := cmd.Run(); err == nil {
			caps.YoutubeDLAvailable = true
			caps.PreferredTool = "youtube-dl"
		}
	}
	
	return caps, nil
}

// YTDLCapabilities represents YouTube downloader capabilities
type YTDLCapabilities struct {
	YTDLPAvailable      bool
	YoutubeDLAvailable  bool
	PreferredTool       string
	SupportedSites      []string
	SupportedFormats    []string
}

// HasYouTubeDownloader returns true if any YouTube downloader is available
func (c *YTDLCapabilities) HasYouTubeDownloader() bool {
	return c.YTDLPAvailable || c.YoutubeDLAvailable
}

// GetBestDownloader returns the best available YouTube downloader
func (c *YTDLCapabilities) GetBestDownloader() string {
	if c.YTDLPAvailable {
		return "yt-dlp"
	}
	if c.YoutubeDLAvailable {
		return "youtube-dl"
	}
	return ""
}