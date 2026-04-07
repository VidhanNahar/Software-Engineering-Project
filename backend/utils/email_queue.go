package utils

import (
	"log"
	"sync"
	"time"
)

// EmailJob represents an email to be sent
type EmailJob struct {
	UserEmail string
	UserName  string
	OTP       string
	Timestamp time.Time
	Attempts  int
}

// EmailQueue manages async email sending with retry logic
type EmailQueue struct {
	jobCh      chan *EmailJob
	wg         sync.WaitGroup
	maxWorkers int
	stopCh     chan struct{}
}

// NewEmailQueue creates a new email queue with background workers
func NewEmailQueue(maxWorkers int, bufferSize int) *EmailQueue {
	eq := &EmailQueue{
		jobCh:      make(chan *EmailJob, bufferSize),
		maxWorkers: maxWorkers,
		stopCh:     make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		eq.wg.Add(1)
		go eq.worker(i)
	}

	log.Printf("📧 Email queue started with %d workers (buffer: %d)", maxWorkers, bufferSize)
	return eq
}

// SubmitJob adds an email job to the queue (non-blocking)
func (eq *EmailQueue) SubmitJob(email, name, otp string) {
	job := &EmailJob{
		UserEmail: email,
		UserName:  name,
		OTP:       otp,
		Timestamp: time.Now(),
		Attempts:  0,
	}

	select {
	case eq.jobCh <- job:
		// Job queued successfully
	default:
		// Queue is full, log warning but don't block request handler
		log.Printf("⚠️ Email queue full, dropping OTP email for %s", email)
	}
}

// worker processes jobs from the queue with retry logic
func (eq *EmailQueue) worker(id int) {
	defer eq.wg.Done()

	for {
		select {
		case <-eq.stopCh:
			log.Printf("📧 Email worker %d shutting down", id)
			return
		case job := <-eq.jobCh:
			if job == nil {
				return
			}
			eq.processJob(job)
		}
	}
}

// processJob sends an email with retry logic (max 3 attempts)
func (eq *EmailQueue) processJob(job *EmailJob) {
	maxRetries := 3
	backoffDuration := 1 * time.Second

	for job.Attempts < maxRetries {
		job.Attempts++

		// Try sending email
		err := SendOTP(job.UserEmail, job.UserName, job.OTP)
		if err == nil {
			log.Printf("✅ Email sent to %s (attempt %d/%d)", job.UserEmail, job.Attempts, maxRetries)
			return
		}

		// If last attempt, give up
		if job.Attempts >= maxRetries {
			log.Printf("❌ Failed to send email to %s after %d attempts: %v", job.UserEmail, maxRetries, err)
			return
		}

		// Wait before retry (exponential backoff)
		log.Printf("⚠️ Email send failed for %s (attempt %d/%d), retrying in %v: %v",
			job.UserEmail, job.Attempts, maxRetries, backoffDuration, err)
		time.Sleep(backoffDuration)
		backoffDuration *= 2
	}
}

// Stop gracefully shuts down the email queue
func (eq *EmailQueue) Stop() {
	log.Println("📧 Stopping email queue...")
	close(eq.stopCh)

	// Wait for all workers to finish (with timeout)
	done := make(chan struct{})
	go func() {
		eq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ Email queue stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("⚠️ Email queue shutdown timeout (some emails may be lost)")
	}
}

// GetQueueStats returns current queue statistics
func (eq *EmailQueue) GetQueueStats() map[string]int {
	return map[string]int{
		"pending": len(eq.jobCh),
		"workers": eq.maxWorkers,
	}
}
