package channelserver

import (
	"fmt"

	"go.uber.org/zap"
)

// AchievementService encapsulates business logic for the achievement system.
type AchievementService struct {
	achievementRepo AchievementRepo
	logger          *zap.Logger
}

// NewAchievementService creates a new AchievementService.
func NewAchievementService(ar AchievementRepo, log *zap.Logger) *AchievementService {
	return &AchievementService{achievementRepo: ar, logger: log}
}

const achievementEntryCount = uint8(33)

// AchievementSummary holds the computed achievements and total points for a character.
type AchievementSummary struct {
	Points       uint32
	Achievements [33]Achievement
}

// GetAll ensures the achievement record exists, fetches all scores, and computes
// the achievement state for every category. Returns the total accumulated points
// and per-category Achievement data.
func (svc *AchievementService) GetAll(charID uint32) (*AchievementSummary, error) {
	if err := svc.achievementRepo.EnsureExists(charID); err != nil {
		svc.logger.Error("Failed to ensure achievements record", zap.Error(err))
	}

	scores, err := svc.achievementRepo.GetAllScores(charID)
	if err != nil {
		return nil, err
	}

	var summary AchievementSummary
	for id := uint8(0); id < achievementEntryCount; id++ {
		ach := GetAchData(id, scores[id])
		summary.Points += ach.Value
		summary.Achievements[id] = ach
	}
	return &summary, nil
}

// Increment validates the achievement ID, ensures the record exists, and bumps
// the score for the given achievement category.
func (svc *AchievementService) Increment(charID uint32, achievementID uint8) error {
	if achievementID > 32 {
		return fmt.Errorf("achievement ID %d out of range [0, 32]", achievementID)
	}

	if err := svc.achievementRepo.EnsureExists(charID); err != nil {
		svc.logger.Error("Failed to ensure achievements record", zap.Error(err))
	}

	return svc.achievementRepo.IncrementScore(charID, achievementID)
}
