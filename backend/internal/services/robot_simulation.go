package services

import (
	"math"
	"stone-relic-monitor/internal/models"
	"time"
)

type RobotSimulationService struct{}

func NewRobotSimulationService() *RobotSimulationService {
	return &RobotSimulationService{}
}

func lerp(a, b float32, t float32) float32 {
	return a + (b-a)*t
}

func (s *RobotSimulationService) Simulate(req *models.RobotSimulationRequest) *models.RobotSimulationResult {
	nPoints := len(req.Path)
	if nPoints == 0 {
		return &models.RobotSimulationResult{
			RelicID:     req.RelicID,
			Frames:      []models.RobotFrame{},
			TotalFrames: 0,
			DurationSec: 0,
			AreaCleaned: 0,
		}
	}

	speedFactor := req.SpeedFactor
	if speedFactor <= 0 {
		speedFactor = 1.0
	}

	framesPerSegment := 15
	totalFrames := framesPerSegment * (nPoints + 1)

	frames := make([]models.RobotFrame, 0, totalFrames)
	areaCleaned := float32(0)

	curPos := [3]float32{req.StartPosition[0], req.StartPosition[1], req.StartPosition[2]}

	for pointIdx, target := range req.Path {
		for i := 0; i < framesPerSegment; i++ {
			t := float32(i) / float32(framesPerSegment)
			pos := [3]float32{
				lerp(curPos[0], target.X, t),
				lerp(curPos[1], target.Y, t),
				lerp(curPos[2], target.Z, t),
			}

			dx := float64(target.X - curPos[0])
			dy := float64(target.Y - curPos[1])
			dz := float64(target.Z - curPos[2])
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
			rotX := float32(0)
			rotY := float32(0)
			if dist > 1e-6 {
				rotY = float32(math.Atan2(dx, dz))
				rotX = float32(math.Atan2(-dy, math.Sqrt(dx*dx+dz*dz)))
			}

			cleaningArea := [][]float32{}
			laserActive := false
			if t > 0.85 && pointIdx < nPoints {
				laserActive = true
				spotRadius := float32(2.5)
				for ang := 0; ang < 16; ang++ {
					angle := float64(ang) / 16 * 2 * math.Pi
					cleaningArea = append(cleaningArea, []float32{
						target.X + float32(math.Cos(angle))*spotRadius,
						target.Y + float32(math.Sin(angle))*spotRadius,
						target.Z,
					})
				}
			}

			progress := float32(pointIdx) / float32(nPoints)

			frames = append(frames, models.RobotFrame{
				Timestamp:      time.Now().UnixNano()/1e6 + int64(i*1000/30/speedFactor),
				RobotPosition:  pos,
				RobotRotation:  [3]float32{rotX, rotY, 0},
				CurrentPointID: target.ID,
				LaserActive:    laserActive,
				CleaningArea:   cleaningArea,
				Progress:       progress,
			})
		}
		curPos = [3]float32{target.X, target.Y, target.Z}
		areaCleaned += float32(math.Pi * 2.5 * 2.5)
	}

	finalProgress := float32(1.0)
	for i := 0; i < framesPerSegment; i++ {
		frames = append(frames, models.RobotFrame{
			Timestamp:      time.Now().UnixNano()/1e6 + int64((framesPerSegment*nPoints+i)*1000/30/speedFactor),
			RobotPosition:  curPos,
			RobotRotation:  [3]float32{0, 0, 0},
			CurrentPointID: -1,
			LaserActive:    false,
			CleaningArea:   [][]float32{},
			Progress:       finalProgress,
		})
	}

	return &models.RobotSimulationResult{
		RelicID:     req.RelicID,
		Frames:      frames,
		TotalFrames: len(frames),
		DurationSec: float32(len(frames)) / 30.0 / speedFactor,
		AreaCleaned: areaCleaned,
	}
}
