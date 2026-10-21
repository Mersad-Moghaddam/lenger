package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// ScannerName identifies the detected scanner tool.
type ScannerName string

const (
	ScannerTrivy ScannerName = "trivy"
	ScannerGrype ScannerName = "grype"
	ScannerScout ScannerName = "scout"
	ScannerNone  ScannerName = "none"
)

// DetectScanner checks PATH for an available container vulnerability scanner (IMG-10).
func DetectScanner() ScannerName {
	if _, err := exec.LookPath("trivy"); err == nil {
		return ScannerTrivy
	}
	if _, err := exec.LookPath("grype"); err == nil {
		return ScannerGrype
	}
	if _, err := exec.LookPath("docker"); err == nil {
		// Verify if docker scout plugin exists
		cmd := exec.Command("docker", "scout", "version")
		if err := cmd.Run(); err == nil {
			return ScannerScout
		}
	}
	return ScannerNone
}

// ScanImage runs the first available scanner against imageRef (IMG-10).
func (s *Service) ScanImage(ctx context.Context, imageRef string) (engine.VulnerabilityReport, error) {
	scanner := DetectScanner()
	if scanner == ScannerNone {
		return engine.VulnerabilityReport{
			Scanner:   "none",
			ScannedAt: time.Now(),
			Error:     "No scanner found in PATH. Install trivy, grype, or docker scout.",
		}, nil
	}

	scanCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	switch scanner {
	case ScannerTrivy:
		return runTrivy(scanCtx, imageRef)
	case ScannerGrype:
		return runGrype(scanCtx, imageRef)
	case ScannerScout:
		return runScout(scanCtx, imageRef)
	default:
		return engine.VulnerabilityReport{
			Scanner:   string(scanner),
			ScannedAt: time.Now(),
		}, nil
	}
}

func runTrivy(ctx context.Context, imageRef string) (engine.VulnerabilityReport, error) {
	cmd := exec.CommandContext(ctx, "trivy", "image", "--quiet", "--format", "json", imageRef)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return engine.VulnerabilityReport{
			Scanner:   "trivy",
			ScannedAt: time.Now(),
			Error:     strings.TrimSpace(stderr.String()),
		}, nil
	}

	return parseTrivyOutput(stdout.Bytes())
}

func parseTrivyOutput(data []byte) (engine.VulnerabilityReport, error) {
	var parsed struct {
		Results []struct {
			Vulnerabilities []struct {
				Severity     string `json:"Severity"`
				FixedVersion string `json:"FixedVersion"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	report := engine.VulnerabilityReport{
		Scanner:   "trivy",
		ScannedAt: time.Now(),
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		report.Error = fmt.Sprintf("parse trivy report: %v", err)
		return report, nil
	}

	for _, res := range parsed.Results {
		for _, v := range res.Vulnerabilities {
			switch strings.ToUpper(v.Severity) {
			case "CRITICAL":
				report.Critical++
			case "HIGH":
				report.High++
			case "MEDIUM":
				report.Medium++
			case "LOW":
				report.Low++
			}
			if v.FixedVersion != "" {
				report.Fixable++
			}
		}
	}
	return report, nil
}

func runGrype(ctx context.Context, imageRef string) (engine.VulnerabilityReport, error) {
	cmd := exec.CommandContext(ctx, "grype", imageRef, "-o", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return engine.VulnerabilityReport{
			Scanner:   "grype",
			ScannedAt: time.Now(),
			Error:     strings.TrimSpace(stderr.String()),
		}, nil
	}

	return parseGrypeOutput(stdout.Bytes())
}

func parseGrypeOutput(data []byte) (engine.VulnerabilityReport, error) {
	var parsed struct {
		Matches []struct {
			Vulnerability struct {
				Severity string `json:"severity"`
				Fix      struct {
					State string `json:"state"`
				} `json:"fix"`
			} `json:"vulnerability"`
		} `json:"matches"`
	}

	report := engine.VulnerabilityReport{
		Scanner:   "grype",
		ScannedAt: time.Now(),
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		report.Error = fmt.Sprintf("parse grype report: %v", err)
		return report, nil
	}

	for _, m := range parsed.Matches {
		switch strings.ToUpper(m.Vulnerability.Severity) {
		case "CRITICAL":
			report.Critical++
		case "HIGH":
			report.High++
		case "MEDIUM":
			report.Medium++
		case "LOW":
			report.Low++
		}
		if m.Vulnerability.Fix.State == "fixed" {
			report.Fixable++
		}
	}
	return report, nil
}

func runScout(ctx context.Context, imageRef string) (engine.VulnerabilityReport, error) {
	cmd := exec.CommandContext(ctx, "docker", "scout", "cves", imageRef, "--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return engine.VulnerabilityReport{
			Scanner:   "scout",
			ScannedAt: time.Now(),
			Error:     strings.TrimSpace(stderr.String()),
		}, nil
	}

	var parsed struct {
		Summary struct {
			Critical int `json:"critical"`
			High     int `json:"high"`
			Medium   int `json:"medium"`
			Low      int `json:"low"`
			Fixable  int `json:"fixable"`
		} `json:"summary"`
	}

	report := engine.VulnerabilityReport{
		Scanner:   "scout",
		ScannedAt: time.Now(),
	}

	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		report.Error = fmt.Sprintf("parse scout report: %v", err)
		return report, nil
	}

	report.Critical = parsed.Summary.Critical
	report.High = parsed.Summary.High
	report.Medium = parsed.Summary.Medium
	report.Low = parsed.Summary.Low
	report.Fixable = parsed.Summary.Fixable
	return report, nil
}
