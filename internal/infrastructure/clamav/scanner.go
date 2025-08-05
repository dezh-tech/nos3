package clamav

import (
	"context"
	"io"
	"nos3/internal/domain/repository/clamav"
	"time"

	"github.com/dutchcoders/go-clamd"

	"nos3/internal/domain/entity"
	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Scanner struct {
	clamd      clamav.ClamdClient
	timeout    time.Duration
	grpcClient grpcRepository.IClient
}

func NewScanner(cfg ScannerConfig, grpcClient grpcRepository.IClient) (*Scanner, error) {
	logger.Info("connecting to ClamAV daemon", "address", cfg.Address)

	clamdClient := clamd.NewClamd(cfg.Address)

	scanner := &Scanner{
		clamd:      clamdClient,
		timeout:    time.Duration(cfg.Timeout) * time.Millisecond,
		grpcClient: grpcClient,
	}

	if err := scanner.ping(); err != nil {
		if _, logErr := grpcClient.AddLog(context.Background(),
			"failed to connect to ClamAV daemon", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	return scanner, nil
}

func (s *Scanner) ping() error {
	return s.clamd.Ping()
}

func (s *Scanner) ScanStream(ctx context.Context, reader io.Reader) (entity.MalwareScanResult, error) {
	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	abort := make(chan bool, 1)

	go func() {
		select {
		case <-scanCtx.Done():
			abort <- true
		case <-ctx.Done():
			abort <- true
		}
	}()

	resultChan, err := s.clamd.ScanStream(reader, abort)
	if err != nil {
		s.logError(ctx, "failed to initiate stream scan", err.Error())

		return entity.MalwareScanResult{
				Status: entity.MalwareScanStatusError,
				Error:  err.Error(),
			}, &MalwareError{
				Code:    ErrorCodeScanFailed,
				Message: "failed to initiate stream scan",
				Details: err.Error(),
			}
	}

	return s.processResults(ctx, resultChan)
}

func (s *Scanner) processResults(ctx context.Context,
	resultChan chan *clamd.ScanResult) (entity.MalwareScanResult, error) {
	var threats []string

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				return s.buildResult(threats), nil
			}

			err := s.handleScanResult(result, &threats)
			if err != nil {
				return entity.MalwareScanResult{
					Status:  entity.MalwareScanStatusError,
					Error:   err.Error(),
					Threats: threats,
				}, err
			}

		case <-ctx.Done():
			return entity.MalwareScanResult{
					Status:  entity.MalwareScanStatusError,
					Error:   "context cancelled",
					Threats: threats,
				}, &MalwareError{
					Code:    ErrorCodeTimeout,
					Message: "scan cancelled",
					Details: "context cancelled",
				}

		case <-time.After(s.timeout):
			return entity.MalwareScanResult{
					Status:  entity.MalwareScanStatusError,
					Error:   "scan took too long",
					Threats: threats,
				}, &MalwareError{
					Code:    ErrorCodeTimeout,
					Message: "scan timeout",
					Details: "scan took too long",
				}
		}
	}
}

func (s *Scanner) handleScanResult(result *clamd.ScanResult, threats *[]string) error {
	switch result.Status {
	case clamd.RES_OK:
		return nil

	case clamd.RES_FOUND:
		*threats = append(*threats, result.Description)

		return nil

	case clamd.RES_ERROR, clamd.RES_PARSE_ERROR:
		return &MalwareError{
			Code:    ErrorCodeScanFailed,
			Message: "scan failed",
			Details: result.Description,
		}
	}

	return nil
}

func (s *Scanner) buildResult(threats []string) entity.MalwareScanResult {
	if len(threats) > 0 {
		return entity.MalwareScanResult{
			Status:  entity.MalwareScanStatusInfected,
			Threats: threats,
		}
	}

	return entity.MalwareScanResult{
		Status:  entity.MalwareScanStatusClean,
		Threats: nil,
	}
}

func (s *Scanner) logError(ctx context.Context, message, details string) {
	logger.Error(message, "details", details)
	if _, logErr := s.grpcClient.AddLog(ctx, message, details); logErr != nil {
		logger.Error("can't send log to manager", "err", logErr)
	}
}
