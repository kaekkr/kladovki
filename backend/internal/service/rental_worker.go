package service

import (
	"context"
	"log"
	"time"
)

const RentalWorkerInterval = 10 * time.Second

func (s *Service) StartRentalWorker(ctx context.Context) {
	ticker := time.NewTicker(RentalWorkerInterval)
	defer ticker.Stop()

	log.Println("Rental worker started")

	// Run immediately on startup.
	s.expireRentals(ctx)

	for {
		select {
		case <-ticker.C:
			s.expireRentals(ctx)

		case <-ctx.Done():
			log.Println("Rental worker stopped")
			return
		}
	}
}

func (s *Service) expireRentals(ctx context.Context) {
	if err := s.repo.ExpireRentals(ctx); err != nil {
		log.Printf("rental worker: failed to expire rentals: %v", err)
	}
}
